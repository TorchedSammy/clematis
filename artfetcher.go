package main

import "github.com/godbus/dbus/v5"

type artFetcher interface{
	getAlbumArt(artist, album, title string, metadata map[string]dbus.Variant) string
}
