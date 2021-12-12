package main

import (
	"fmt"
	"os"
	"path/filepath"
	"net/url"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/Pauloo27/go-mpris"
	"github.com/hugolgst/rich-go/client"
)

var pbStat string

func main() {
	// watcher conn is for eavesdropping messages and is a monitor
	// infoConn is for getting other data at any time
	// we can't Call on the watcher conn (error of EOF), so we use the separate infoConn
	watcherConn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	infoConn, _ := dbus.ConnectSessionBus()

	names, err := mpris.List(infoConn)
	if err != nil {
		panic(err)
	}

	if len(names) == 0 {
		fmt.Println("No MPRIS player found.")
		os.Exit(1)
	}

	playerName := names[0]
	player := mpris.New(infoConn, playerName)
	playerIdentity, _ := player.GetIdentity()
	fmt.Println("Getting information from", playerIdentity)

	var rules = []string{
		"type='signal',member='PropertiesChanged',path='/org/mpris/MediaPlayer2',interface='org.freedesktop.DBus.Properties'",
		"type='signal',member='Seeked',path='/org/mpris/MediaPlayer2',interface='org.mpris.MediaPlayer2.Player'",
		"member='NameLost'",
	}

	var flag uint = 0

	initialMetadata, elapsed, pbst, playerConnName := getInitialData(player, infoConn, playerName)
	pbStat = pbst // go is dumb

	call := watcherConn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, flag)
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
	setPresence(initialMetadata, time.Now().Add(-time.Duration(elapsed) * time.Microsecond), player)
	mdata := &initialMetadata

	c := make(chan *dbus.Message, 10)
	watcherConn.Eavesdrop(c)
	for msg := range c {
		msgMember := msg.Headers[dbus.FieldMember].Value().(string)
		fmt.Println(msg.Body)
		// only check if seek is from our main player
		if msgMember == "Seeked" && msg.Headers[dbus.FieldSender].Value().(string) == playerConnName {
			// if the "seek" is at the beginning of the song
			// we dont want to update the presence
			// because a seek with a value of 0 is sent when a song starts,
			// this is to prevent a triple update of the presence
			// playing status will be sent when the song restarts
			if msg.Body[0].(int64) != 0 {
				fmt.Println("Player seeked")
				elapsed, _ := player.GetPosition()
				setPresence(*mdata, time.Now().Add(-time.Duration(elapsed) * time.Second), player)
			}		
		} else if msgMember == "NameLost" {
			if msg.Body[0] == playerName {
				fmt.Println("Main player disconnected")
				// if there was another player connected to dbus
				names, _:= mpris.List(infoConn)
				if len(names) != 0 {
					playerName = names[0]
					player = mpris.New(infoConn, playerName)
					playerIdentity, _ = player.GetIdentity()
					fmt.Println("Switched to", playerIdentity)
					*mdata, elapsed, pbStat, playerConnName = getInitialData(player, infoConn, playerName)
					setPresence(*mdata, time.Now().Add(-time.Duration(elapsed) * time.Microsecond), player)
				} else {
					client.Logout()
				}
			}
		}

		if len(msg.Body) <= 1 || msg.Headers[dbus.FieldSender].Value().(string) != playerConnName {
			continue
		}

		bodyMap := msg.Body[1].(map[string]dbus.Variant)
		metadata := getMetadata(bodyMap)
		if metadata != nil {
			mdata = metadata
		}
		setPresence(*mdata, time.Now(), player)
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

func setPresence(metadata map[string]dbus.Variant, songstamp time.Time, player *mpris.Player) {
	var startstamp *time.Time
	var endstamp *time.Time
	if songLength, ok := metadata["mpris:length"].Value().(int64); ok {
		stampTime := songstamp.Add(time.Duration(songLength) * time.Microsecond)
		startstamp = &songstamp
		endstamp = &stampTime
	}
	pbStat, _ := player.GetPlaybackStatus()
	playerIdentity, _ := player.GetIdentity()

	title := ""
	if songtitle, ok := metadata["xesam:title"].Value().(string); ok {
		title = songtitle
	} else {
		title = "Playing..." // i guess if we cant get the filename or proper title this works?
		if titleUrlEscaped, ok := metadata["xesam:url"].Value().(string); ok {
			title, _ = url.PathUnescape(filepath.Base(titleUrlEscaped))
		}
	}
	album := ""
	if abm, ok := metadata["xesam:album"].Value().(string); ok {
		album = " on " + abm
	}
	if pbStat != "Playing" {
		startstamp, endstamp = nil, nil
	}

	artistsStr := ""
	if artistsArr, ok := metadata["xesam:artist"].Value().([]string); ok {
		artistsStr = "by " + strings.Join(artistsArr, ", ")
	}
	client.SetActivity(client.Activity{
		Details: title,
		State: artistsStr + album,
		LargeImage: "music",
		LargeText: playerIdentity,
		SmallImage: strings.ToLower(string(pbStat)),
		SmallText: string(pbStat),
		Timestamps: &client.Timestamps{
			Start: startstamp,
			End: endstamp,
		},
	})
}

func getInitialData(player *mpris.Player, conn *dbus.Conn, playerName string) (map[string]dbus.Variant, float64, string, string) {
	pbStat, _ := player.GetPlaybackStatus()
	playerPos, _ := player.GetPosition()
	data, _ := player.GetMetadata()
	playerconnVariant := dbus.Variant{}
	conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").Call("org.freedesktop.DBus.GetNameOwner", 0, playerName).Store(&playerconnVariant)
	playerConnName := playerconnVariant.Value().(string)
	return data, playerPos * 1000000, string(pbStat), playerConnName
}

