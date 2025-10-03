package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

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
