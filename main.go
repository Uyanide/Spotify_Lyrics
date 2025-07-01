package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

func getCacheDir() (string, error) {
	var dir string
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log(fmt.Sprintf("Error getting user cache directory: %v", err))
		dir = filepath.Join(os.TempDir(), "spotify_lyrics")
	} else {
		dir = filepath.Join(cacheDir, "spotify_lyrics")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("error creating cache directory: %v", err)
	}
	return dir, nil
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
	argNumLines   int
	argOutputPath string
	argOffset     int
	argOffsetFile string
	argInterval   int
	argAhead      int
	argCls        bool
	argPureOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "spotify-lyrics",
	Short: "A tool to fetch and display Spotify lyrics",
	Long:  "A command-line tool to fetch and display Spotify lyrics with caching support.",
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch lyrics for current track",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir, err := getCacheDir()
		if err != nil {
			log(fmt.Sprintf("Error initializing cache directory: %v", err))
			return
		}

		res, err := fetchLyrics(cacheDir)
		if err != nil || res == nil || res.IsError {
			log(err.Error())
			return
		}

		if argPureOutput {
			for _, lyric := range res.Lyrics {
				fmt.Println(lyric.Words)
			}
		} else {
			res.lrcEncodeFile("/dev/stdout")
		}
	},
}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen mode - continuously display lyrics",
	Run: func(cmd *cobra.Command, args []string) {
		if argNumLines < 1 {
			log("Number of lines must be positive, correcting to 1")
			argNumLines = 1
		}
		if argAhead < 0 {
			log("Ahead lines must be non-negative, correcting to 0")
			argAhead = 0
		}
		cacheDir, err := getCacheDir()
		if err != nil {
			log(fmt.Sprintf("Error initializing cache directory: %v", err))
			return
		}
		lockFile := filepath.Join(cacheDir, "spotify-lyrics.lock")
		listen(argNumLines, cacheDir, argOutputPath, lockFile, argOffset, argOffsetFile, argInterval, argAhead, argCls)
	},
}

var printCmd = &cobra.Command{
	Use:   "print",
	Short: "Print mode - single shot display",
	Run: func(cmd *cobra.Command, args []string) {
		if argNumLines < 1 {
			log("Number of lines must be positive, correcting to 1")
			argNumLines = 1
		}
		if argAhead < 0 {
			log("Ahead lines must be non-negative, correcting to 0")
			argAhead = 0
		}
		cacheDir, err := getCacheDir()
		if err != nil {
			log(fmt.Sprintf("Error initializing cache directory: %v", err))
			return
		}
		print(argNumLines, cacheDir, argOutputPath, argOffset, argOffsetFile, argAhead, argCls)
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear [trackID]",
	Short: "Clear all cached lyrics or for a specific track",
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir, err := getCacheDir()
		if err != nil {
			log(fmt.Sprintf("Error initializing cache directory: %v", err))
			return
		}
		if len(args) > 0 {
			trackID := args[0]
			trackFile := filepath.Join(cacheDir, trackID+".lrc")
			if err := os.Remove(trackFile); err != nil {
				log(fmt.Sprintf("Error removing track cache file: %v", err))
				return
			}
			log(fmt.Sprintf("Cache for track ID %s cleared", trackID))
		} else if err := os.RemoveAll(cacheDir); err != nil {
			log(fmt.Sprintf("Error clearing cache directory: %v", err))
		} else {
			log("Cache directory cleared")
		}
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
		trackInfo := getTrackDisplayTitle()
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get if the current track is playing",
	Run: func(_ *cobra.Command, _ []string) {
		status, err := getPlayingStatus()
		if err != nil || !status {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	},
}

func init() {
	// Fetch command flags
	fetchCmd.Flags().BoolVarP(&argPureOutput, "pure", "p", false, "Output lyrics without times")

	// Listen/Print command flags
	listenCmd.Flags().IntVarP(&argNumLines, "lines", "l", 5, "Number of lines to display")
	listenCmd.Flags().StringVarP(&argOutputPath, "output", "o", "/dev/stdout", "Output file path")
	listenCmd.Flags().StringVarP(&argOffsetFile, "offset-file", "f", "", "File to read offset from (if not set, uses --offset)")
	listenCmd.Flags().IntVarP(&argOffset, "offset", "O", 0, "Offset in milliseconds for lyrics timing (ignored if --offset-file is set)")
	listenCmd.Flags().IntVarP(&argInterval, "interval", "i", 200, "Interval in milliseconds beteen updates")
	listenCmd.Flags().IntVarP(&argAhead, "ahead", "a", 0, "Number of lines to display ahead of current position")
	listenCmd.Flags().BoolVarP(&argCls, "cls", "c", false, "Clear the terminal before displaying lyrics")

	printCmd.Flags().IntVarP(&argNumLines, "lines", "l", 5, "Number of lines to display")
	printCmd.Flags().StringVarP(&argOutputPath, "output", "o", "/dev/stdout", "Output file path")
	printCmd.Flags().StringVarP(&argOffsetFile, "offset-file", "f", "", "File to read offset from (if not set, uses --offset)")
	printCmd.Flags().IntVarP(&argOffset, "offset", "O", 0, "Offset in milliseconds for lyrics timing (ignored if --offset-file is set)")
	printCmd.Flags().IntVarP(&argAhead, "ahead", "a", 0, "Number of lines to display ahead of current position")
	printCmd.Flags().BoolVarP(&argCls, "cls", "c", false, "Clear the terminal before displaying lyrics")

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
	rootCmd.AddCommand(statusCmd)
}

func main() {
	defer closeDBus()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
