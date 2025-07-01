package main

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	spotifyBusName  = "org.mpris.MediaPlayer2.spotify"
	mprisPath       = "/org/mpris/MediaPlayer2"
	playerInterface = "org.mpris.MediaPlayer2.Player"
)

var conn *dbus.Conn

func initDBus() error {
	if conn != nil {
		return nil
	}

	var err error
	conn, err = dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %v", err)
	}
	return nil
}

func closeDBus() {
	if conn != nil {
		conn.Close()
		conn = nil
	}
}

func getMetadata[T any](key string) (T, error) {
	var zero T // default value

	if err := initDBus(); err != nil {
		return zero, err
	}

	obj := conn.Object(spotifyBusName, mprisPath)

	var metadata map[string]dbus.Variant
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0, playerInterface, "Metadata").Store(&metadata)
	if err != nil {
		return zero, fmt.Errorf("error getting metadata: %v", err)
	}

	value, exists := metadata[key]
	if !exists {
		return zero, fmt.Errorf("key %s not found in metadata", key)
	}

	var result T
	if err := value.Store(&result); err != nil {
		return zero, fmt.Errorf("error storing value for key %s: %v", key, err)
	}
	return result, nil
}

func getTrackID() (string, error) {
	trackID, err := getMetadata[string]("mpris:trackid")
	if err != nil {
		return "", err
	}

	parts := strings.Split(trackID, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid track ID format")
	}

	return parts[len(parts)-1], nil
}

func getPosition() (int, error) {
	if err := initDBus(); err != nil {
		return -1, err
	}

	obj := conn.Object(spotifyBusName, mprisPath)

	var position uint64
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0, playerInterface, "Position").Store(&position)
	if err != nil {
		return -1, fmt.Errorf("error getting position: %v", err)
	}

	return int(position / 1000), nil // Convert microseconds to milliseconds
}

func getArtist() (string, error) {
	artist, err := getMetadata[[]string]("xesam:artist")
	if err != nil {
		return "", fmt.Errorf("error getting artist: %v", err)
	}
	return strings.Join(artist, ", "), nil
}

func getTitle() (string, error) {
	title, err := getMetadata[string]("xesam:title")
	if err != nil {
		return "", fmt.Errorf("error getting title: %v", err)
	}
	return title, nil
}

func getTrackDisplayTitle() string {
	artist, err := getArtist()
	if err != nil {
		artist = "UNKOWN ARTIST"
	}

	title, err := getTitle()
	if err != nil {
		title = "UNKOWN TITLE"
	}

	return fmt.Sprintf("%s - %s", artist, title)
}

func getAlbum() (string, error) {
	album, err := getMetadata[string]("xesam:album")
	if err != nil {
		return "", fmt.Errorf("error getting album: %v", err)
	}
	return album, nil
}

func getLength() (int, error) {
	length, err := getMetadata[uint64]("mpris:length")
	if err != nil {
		return 0, fmt.Errorf("error getting track length: %v", err)
	}

	return int(length / 1000), nil // Convert microseconds to milliseconds
}

func setPosition(position int) error {
	if err := initDBus(); err != nil {
		return err
	}

	obj := conn.Object(spotifyBusName, mprisPath)

	fullTrackID, err := getMetadata[string]("mpris:trackid")
	if err != nil {
		return fmt.Errorf("error getting track ID: %v", err)
	}

	positionMicroseconds := int64(position * 1000) // Convert milliseconds to microseconds and use int64

	// Use the full track ID as object path
	call := obj.Call(playerInterface+".SetPosition", 0, dbus.ObjectPath(fullTrackID), positionMicroseconds)
	if call.Err != nil {
		return fmt.Errorf("error setting position: %v", call.Err)
	}

	return nil
}

func getPlayingStatus() (bool, error) {
	if err := initDBus(); err != nil {
		return false, err
	}

	obj := conn.Object(spotifyBusName, mprisPath)

	var status string
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0, playerInterface, "PlaybackStatus").Store(&status)
	if err != nil {
		return false, fmt.Errorf("error getting playback status: %v", err)
	}

	return status == "Playing", nil
}
