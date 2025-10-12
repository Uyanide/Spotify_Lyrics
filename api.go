// Ref: [spotify-lyrics-api](https://github.com/akashrchandran/spotify-lyrics-api)

package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

type SpotifySecret struct {
	Version int    `json:"version"`
	Secret  string `json:"secret"`
}

type SpotifySecrets []SpotifySecret

var (
	err404 = fmt.Errorf("no lyrics found (404)")
)

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
	if SP_DC == "" {
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
	req.Header.Set("Cookie", "sp_dc="+SP_DC)

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

	secret, version, err := getLatestSecretKeyVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret key: %w", err)
	}

	totpCode, err := generateTOTP(serverTimeData.ServerTime, secret)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("reason", "transport")
	params.Set("productType", "web-player")
	params.Set("totp", totpCode)
	params.Set("totpVer", strconv.Itoa(version))
	params.Set("ts", strconv.FormatInt(time.Now().Unix(), 10))

	return params, nil
}

// generates TOTP using the server time and provided secret
func generateTOTP(serverTimeSeconds int64, secret string) (string, error) {
	period := int64(30)
	digits := 6

	counter := uint64(serverTimeSeconds / period)
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)

	mac := hmac.New(sha1.New, []byte(secret))
	if _, err := mac.Write(counterBytes[:]); err != nil {
		return "", fmt.Errorf("failed to write hmac: %w", err)
	}
	digest := mac.Sum(nil)

	if len(digest) < 4 {
		return "", fmt.Errorf("hmac digest too short")
	}
	offset := int(digest[len(digest)-1] & 0x0F)
	// dynamic truncation
	binaryCode := (int(digest[offset])&0x7F)<<24 |
		(int(digest[offset+1])&0xFF)<<16 |
		(int(digest[offset+2])&0xFF)<<8 |
		(int(digest[offset+3]) & 0xFF)

	// compute modulus 10^digits without importing math
	mod := 1
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	code := binaryCode % mod

	// zero-pad to digits
	return fmt.Sprintf("%0*d", digits, code), nil
}

// Fetches the latest secret and its version
func getLatestSecretKeyVersion() (string, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(SECRET_KEY_URL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch secret: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("secret endpoint returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read secret response: %w", err)
	}

	var arr SpotifySecrets
	if err := json.Unmarshal(body, &arr); err != nil {
		return "", 0, fmt.Errorf("invalid secret json: %w", err)
	}
	if len(arr) == 0 {
		return "", 0, fmt.Errorf("secret json empty")
	}
	last := arr[len(arr)-1]
	secretVal := last.Secret
	versionVal := last.Version

	parts := make([]string, 0, len(secretVal))
	for i, r := range secretVal {
		transformed := int(r) ^ ((i % 33) + 9)
		parts = append(parts, strconv.Itoa(transformed))
	}
	transformedStr := strings.Join(parts, "")
	return transformedStr, versionVal, nil
}

func getLyrics(trackID string) (*LyricsResponse, error) {
	token, err := getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	client := &http.Client{Timeout: FETCH_TIMEOUT}

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
		if resp.StatusCode == http.StatusNotFound {
			return nil, err404
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

func (data *LyricsData) fetchLyricsSpotify() error {
	resp, err := getLyrics(data.TrackID)
	if err != nil {
		if err == err404 {
			data.Is404 = true
			return nil
		}
		return err
	}
	data.IsLineSynced = resp.Lyrics.SyncType == "LINE_SYNCED"

	for _, line := range resp.Lyrics.Lines {
		ms, err := strconv.Atoi(line.StartTimeMs)
		if err != nil {
			log(fmt.Sprintf("Error parsing time tag '%s': %v", line.StartTimeMs, err))
			continue // skip this line if parsing fails
		}
		data.Lyrics = append(data.Lyrics, LyricLine{
			StartTimeMs: ms,
			Words:       line.Words,
		})
	}

	return nil
}
