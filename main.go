package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var config *Config

func init() {
	config = LoadConfig()
}

func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log(fmt.Sprintf("Error getting home directory: %v", err))
		return "/tmp/eww/lyrics"
	}
	dir := filepath.Join(homeDir, ".cache", "eww", "lyrics")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log(fmt.Sprintf("Error creating cache directory: %v", err))
		return "/tmp/eww/lyrics"
	}
	return dir
}

func acquireLock(lockFile string) (*os.File, error) {
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("another instance is already running")
	}

	fmt.Fprintf(file, "%d", os.Getpid())
	file.Sync()

	return file, nil
}

var (
	argMumLines   int
	argOutputPath string
	argTrackID    string
	argOffset     int
	argOffsetFile string
	argInterval   int
	argPureOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "spotify-lyrics",
	Short: "A tool to fetch and display Spotify lyrics",
	Long:  "A command-line tool to fetch and display Spotify lyrics with caching support.",
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch lyrics for current or specified track",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir := getCacheDir()
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			log(fmt.Sprintf("Error creating cache directory: %v", err))
			return
		}

		var finalTrackID string
		if len(args) > 0 {
			finalTrackID = args[0]
		} else if argTrackID != "" {
			finalTrackID = argTrackID
		} else {
			var err error
			finalTrackID, err = getTrackID()
			if err != nil {
				log(fmt.Sprintf("Error getting track ID: %v", err))
				return
			}
		}

		res, err := fetchLyrics(finalTrackID, cacheDir)
		if err != nil || res == nil || res.IsInvalid {
			log(err.Error())
			return
		}

		if argPureOutput {
			for _, lyric := range res.Lyrics {
				fmt.Println(lyric.Lyric)
			}
		} else {
			for _, lyric := range res.Lyrics {
				fmt.Printf("%d %s\n", lyric.Time, lyric.Lyric)
			}
		}
	},
}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen mode - continuously display lyrics",
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir := getCacheDir()
		lockFile := filepath.Join(cacheDir, "spotify-lyrics.lock")
		listen(argMumLines, cacheDir, argOutputPath, lockFile, argOffset, argOffsetFile, argInterval)
	},
}

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Print mode - single shot display",
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir := getCacheDir()
		print(argMumLines, cacheDir, argOutputPath, argOffset, argOffsetFile)
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the lyrics cache",
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir := getCacheDir()
		if err := os.RemoveAll(cacheDir); err != nil {
			log(fmt.Sprintf("Error clearing cache directory: %v", err))
			return
		}
		log("Cache directory cleared")
	},
}

var lengthCmd = &cobra.Command{
	Use:   "length",
	Short: "Get the length of the current track (in ms)",
	Run: func(_ *cobra.Command, _ []string) {
		length, err := getLength()
		if err != nil {
			log(fmt.Sprintf("Error getting track length: %v", err))
			return
		}
		fmt.Printf("%d\n", length)
	},
}

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "Get the current position of the track (in ms)",
	Run: func(cmd *cobra.Command, args []string) {
		position, err := getPosition()
		if err != nil {
			log(fmt.Sprintf("Error getting track position: %v", err))
			return
		}
		fmt.Printf("%d\n", position)
	},
}

var setPositionCmd = &cobra.Command{
	Use:   "set-position [position]",
	Short: "Set the current position of the track (in ms)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		position, err := strconv.Atoi(args[0])
		if err != nil {
			log(fmt.Sprintf("Invalid position: %v", err))
			return
		}
		if err := setPosition(position); err != nil {
			log(fmt.Sprintf("Error setting track position: %v", err))
			return
		}
		log(fmt.Sprintf("Track position set to: %d ms", position))
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get information about the current track",
	Run: func(_ *cobra.Command, _ []string) {
		trackInfo, err := getTrackInfo()
		if err != nil {
			log(fmt.Sprintf("Error getting track info: %v", err))
			return
		}
		fmt.Println(trackInfo)
	},
}

var trackIDCmd = &cobra.Command{
	Use:   "trackid",
	Short: "Get the current track ID",
	Run: func(_ *cobra.Command, _ []string) {
		trackID, err := getTrackID()
		if err != nil {
			log(fmt.Sprintf("Error getting track ID: %v", err))
			return
		}
		fmt.Println(trackID)
	},
}

func init() {
	// Fetch command flags
	fetchCmd.Flags().StringVarP(&argTrackID, "track", "t", "", "Track ID to fetch")
	fetchCmd.Flags().BoolVarP(&argPureOutput, "pure", "p", false, "Output lyrics without times")

	// Listen/Print command flags
	listenCmd.Flags().IntVarP(&argMumLines, "lines", "l", 5, "Number of lines to display")
	listenCmd.Flags().StringVarP(&argOutputPath, "output", "o", "/dev/stdout", "Output file path")
	listenCmd.Flags().StringVarP(&argOffsetFile, "offset-file", "f", "", "File to read offset from (if not set, uses --offset)")
	listenCmd.Flags().IntVarP(&argOffset, "offset", "O", 0, "Offset in milliseconds for lyrics timing (ignored if --offset-file is set)")
	listenCmd.Flags().IntVarP(&argInterval, "interval", "i", 200, "Interval in milliseconds beteen updates")

	printCmd.Flags().IntVarP(&argMumLines, "lines", "l", 5, "Number of lines to display")
	printCmd.Flags().StringVarP(&argOutputPath, "output", "o", "/dev/stdout", "Output file path")
	printCmd.Flags().StringVarP(&argOffsetFile, "offset-file", "f", "", "File to read offset from (if not set, uses --offset)")
	printCmd.Flags().IntVarP(&argOffset, "offset", "O", 0, "Offset in milliseconds for lyrics timing (ignored if --offset-file is set)")

	// Add commands to root
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(listenCmd)
	rootCmd.AddCommand(printCmd)
	rootCmd.AddCommand(clearCmd)
	rootCmd.AddCommand(lengthCmd)
	rootCmd.AddCommand(positionCmd)
	rootCmd.AddCommand(setPositionCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(trackIDCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
