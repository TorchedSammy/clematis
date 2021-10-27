package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/TorchedSammy/go-mpris"
	"github.com/hugolgst/rich-go/client"
)

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	conn2, _ := dbus.ConnectSessionBus()

	names, err := mpris.List(conn)
	if err != nil {
		panic(err)
	}

	if len(names) == 0 {
		fmt.Println("No MPRIS player found.")
		os.Exit(1)
	}

	name := names[0]
	player := mpris.New(conn, name)
	fmt.Println("Getting information from", player.GetIdentity())

	var rules = []string{
		"type='signal',member='PropertiesChanged',path='/org/mpris/MediaPlayer2',interface='org.freedesktop.DBus.Properties'",
		"type='signal',member='Seeked',path='/org/mpris/MediaPlayer2',interface='org.mpris.MediaPlayer2.Player'",
	}

	var flag uint = 0

	data := dbus.Variant{}
	elapsedFromDbus := dbus.Variant{}
	playbackstat := dbus.Variant{}
	conn.Object(name, "/org/mpris/MediaPlayer2").Call("org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "Metadata").Store(&data)
	conn.Object(name, "/org/mpris/MediaPlayer2").Call("org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "Position").Store(&elapsedFromDbus)
	conn.Object(name, "/org/mpris/MediaPlayer2").Call("org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "PlaybackStatus").Store(&playbackstat)
	initialMetadata, elapsed := data.Value().(map[string]dbus.Variant), elapsedFromDbus.Value().(int64)
	initialMetadata["pbStat"] = playbackstat

	call := conn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, flag)
	if call.Err != nil {
		fmt.Fprintln(os.Stderr, "Failed to become monitor:", call.Err)
		os.Exit(1)
	}

	err = client.Login("902662551119224852")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// the reason why we add a negative Duration is because
	// time.Sub returns a Duration but this function needs a Time for a timestamp
	// confusing huh? thanks go
	setPresence(initialMetadata, time.Now().Add(-time.Duration(elapsed) * time.Microsecond))
	c := make(chan *dbus.Message, 10)
	mdata := &initialMetadata
	conn.Eavesdrop(c)
	for msg := range c {
		msgMember := msg.Headers[dbus.FieldMember].Value().(string)
		fmt.Println(msg.Body)
		if msgMember == "Seeked" {
			if msg.Body[0] != 0 {
				conn2.Object(name, "/org/mpris/MediaPlayer2").Call("org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "Position").Store(&elapsedFromDbus)
				elapsed := elapsedFromDbus.Value().(int64)
				setPresence(*mdata, time.Now().Add(-time.Duration(elapsed) * time.Microsecond))
			}		
		}
		if len(msg.Body) <= 1 {
			continue
		}
		bodyMap := msg.Body[1].(map[string]dbus.Variant)
		metadata := getMetadata(bodyMap)
		if metadata != nil {
			mdataSource := *mdata
			for k, v := range *metadata {
				mdataSource[k] = v
			}
		}
		if bodyMap["PlaybackStatus"].Value() != nil {
			mdataSource := *mdata
			mdataSource["pbStat"] = bodyMap["PlaybackStatus"]
		}
		setPresence(*mdata, time.Now())
	}
}

func getMetadata(bodyMap map[string]dbus.Variant) *map[string]dbus.Variant {
	metadataValue := bodyMap["Metadata"].Value()
	if metadataValue == nil {
		return nil
	}
	metadataMap := metadataValue.(map[string]dbus.Variant)

	return &metadataMap
}

func setPresence(metadata map[string]dbus.Variant, songstamp time.Time) {
	songLength := metadata["mpris:length"].Value().(int64)
	stampTime := songstamp.Add(time.Duration(songLength) * time.Microsecond)
	startstamp := &songstamp
	endstamp := &stampTime

	pbStat := metadata["pbStat"].Value().(string)
	if pbStat != "Playing" {
		startstamp, endstamp = nil, nil
	}

	artists := strings.Join(metadata["xesam:artist"].Value().([]string), ", ")
	client.SetActivity(client.Activity{
		Details: metadata["xesam:title"].Value().(string),
		State: "on " + metadata["xesam:album"].Value().(string) + " by " + artists,
		LargeImage: "music",
		LargeText: "cmus",
		SmallImage: strings.ToLower(pbStat),
		SmallText: pbStat,
		Timestamps: &client.Timestamps{
			Start: startstamp,
			End: endstamp,
		},
	})
}
