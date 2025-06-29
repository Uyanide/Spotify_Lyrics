package main

import (
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	TOKEN_URL       = "https://open.spotify.com/api/token"
	LYRICS_URL      = "https://spclient.wg.spotify.com/color-lyrics/v2/track/"
	SERVER_TIME_URL = "https://open.spotify.com/api/server-time"
	USER_AGENT      = "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0" // some random UA from my current browser :)
)

type TokenResponse struct {
	AccessTokenExpirationTimestampMs int64  `json:"accessTokenExpirationTimestampMs"`
	AccessToken                      string `json:"accessToken"`
	IsAnonymous                      bool   `json:"isAnonymous"`
}

type ServerTimeResponse struct {
	ServerTime int64 `json:"serverTime"`
}

type LyricsResponse struct {
	Lyrics struct {
		SyncType string `json:"syncType"`
		Lines    []struct {
			StartTimeMs string   `json:"startTimeMs"`
			Words       string   `json:"words"`
			Syllables   []string `json:"syllables"`
			EndTimeMs   string   `json:"endTimeMs"`
		} `json:"lines"`
	} `json:"lyrics"`
}

func getTokenCacheFile() (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "spotify_token.json"), nil
}

func getFetchLogFile() (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "fetch.log"), nil
}

// will be called each time a new fetch is performed
func appendFetchLog(trackid string, msg string) error {
	logFile, err := getFetchLogFile()
	if err != nil {
		return fmt.Errorf("error getting fetch log file: %w", err)
	}
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening fetch log file: %w", err)
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s [%s] %s\n", timestamp, trackid, msg)
	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("error writing to fetch log file: %w", err)
	}
	return nil
}

// return a valid TokenResponse or nil if error occurs or token expires or whatever
func checkTokenValid() *TokenResponse {
	tokenFile, err := getTokenCacheFile()
	if err != nil {
		log("Error getting token cache file: " + err.Error())
		return nil
	}
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		log("Token cache file does not exist")
		return nil
	}
	content, err := os.ReadFile(tokenFile)
	if err != nil {
		log("Error reading token cache file: " + err.Error())
		return nil
	}
	var tokenData TokenResponse
	if err := json.Unmarshal(content, &tokenData); err != nil {
		log("Error parsing token cache file: " + err.Error())
		return nil
	}
	timeNow := time.Now().UnixMilli()
	if tokenData.AccessTokenExpirationTimestampMs <= timeNow {
		log("Token has expired")
		return nil
	}
	log("Token is valid")
	return &tokenData
}

func writeTokenCache(tokenData *TokenResponse) error {
	tokenFile, err := getTokenCacheFile()
	if err != nil {
		return fmt.Errorf("error getting token cache file: %w", err)
	}
	content, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("error marshaling token data: %w", err)
	}
	if err := os.WriteFile(tokenFile, content, 0644); err != nil {
		return fmt.Errorf("error writing token cache file: %w", err)
	}
	log("Token cache file written successfully")
	return nil
}

func getToken() (string, error) {
	// check cache
	tokenData := checkTokenValid()
	if tokenData != nil {
		return tokenData.AccessToken, nil
	}

	// check SP_DC
	if config.SP_DC == "" {
		return "", fmt.Errorf("SP_DC is not set")
	}

	// build request parameters
	params, err := buildTokenRequestParams()
	if err != nil {
		return "", fmt.Errorf("failed to get server time params: %w", err)
	}

	client := &http.Client{Timeout: 600 * time.Second}
	reqURL := TOKEN_URL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("Cookie", "sp_dc="+config.SP_DC)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status code: %d", resp.StatusCode)
	}

	var tokenJSON TokenResponse
	if err := json.Unmarshal(body, &tokenJSON); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	// don't know if even possible
	if tokenJSON.AccessTokenExpirationTimestampMs <= time.Now().UnixMilli() {
		return "", fmt.Errorf("invalid token response")
	}

	// also cache anonymous tokens to avoid high freq requests when SP_DC is incorrect
	if err := writeTokenCache(&tokenJSON); err != nil {
		return "", fmt.Errorf("failed to write token cache: %w", err)
	}
	log("Token fetched and cached successfully")

	if tokenJSON.IsAnonymous {
		log("Token is anonymous, maybe caused by invalid SP_DC")
		// and ignore
	}

	return tokenJSON.AccessToken, nil
}

// builds request parameters for the token request
func buildTokenRequestParams() (url.Values, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(SERVER_TIME_URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server time: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read server time response: %w", err)
	}

	var serverTimeData ServerTimeResponse
	if err := json.Unmarshal(body, &serverTimeData); err != nil {
		return nil, fmt.Errorf("invalid server time response: %w", err)
	}

	totp, err := generateTOTP(serverTimeData.ServerTime)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("reason", "transport")
	params.Set("productType", "web-player")
	params.Set("totp", totp)
	params.Set("totpVer", "5")
	params.Set("ts", strconv.FormatInt(serverTimeData.ServerTime, 10))

	return params, nil
}

// generates TOTP using the server time
func generateTOTP(serverTimeSeconds int64) (string, error) {
	secretCipher := []int{12, 56, 76, 33, 88, 44, 88, 33, 78, 78, 11, 66, 22, 22, 55, 69, 54}
	processed := make([]int, len(secretCipher))

	for i, b := range secretCipher {
		processed[i] = b ^ (i%33 + 9)
	}

	processedStr := ""
	for _, p := range processed {
		processedStr += fmt.Sprintf("%d", p)
	}

	utf8Bytes := []byte(processedStr)
	hexStr := hex.EncodeToString(utf8Bytes)
	cleanedHex := cleanHex(hexStr)

	secretBytes, err := hex.DecodeString(cleanedHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex string: %w", err)
	}

	secretBase32 := strings.TrimRight(base32.StdEncoding.EncodeToString(secretBytes), "=")

	code, err := totp.GenerateCodeCustom(secretBase32, time.Unix(serverTimeSeconds, 0), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: otp.AlgorithmSHA1,
	})

	return code, err
}

// removes non-hexadecimal characters from a string and ensures it has an even length
func cleanHex(hexStr string) string {
	reg := regexp.MustCompile("[^0123456789abcdefABCDEF]")
	cleaned := reg.ReplaceAllString(hexStr, "")

	if len(cleaned)%2 != 0 {
		cleaned = cleaned[:len(cleaned)-1]
	}
	return cleaned
}

func getLyrics(trackID string) (*LyricsResponse, error) {
	token, err := getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	reqUrl := LYRICS_URL + trackID + "?format=json&market=from_token"
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", USER_AGENT)
	req.Header.Set("App-platform", "WebPlayer")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// special case for 404.
		// which usually means the track has no lyrics or the trackID is invalid,
		// so such result can be cached later to avoid meaningless high freq requests
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("404")
		}
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read lyrics response: %w", err)
	}

	var lyricsResp LyricsResponse
	if err := json.Unmarshal(body, &lyricsResp); err != nil {
		return nil, fmt.Errorf("failed to parse lyrics response: %w", err)
	}

	return &lyricsResp, nil
}
