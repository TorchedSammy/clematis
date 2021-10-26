package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/TorchedSammy/go-mpris"
	"github.com/hugolgst/rich-go/client"
)

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}

	names, err := mpris.List(conn)
	if err != nil {
		panic(err)
	}
	if len(names) == 0 {
		fmt.Println("No media player found.")
		os.Exit(1)
	}

	name := names[0]
	fmt.Println("Found media player:", name)

	player := mpris.New(conn, name)

	fmt.Println("Media player identity:", player.GetIdentity())

	done := make(chan bool, 1)
	var rules = []string{
		"type='signal',member='PropertiesChanged',path='/org/mpris/MediaPlayer2',interface='org.freedesktop.DBus.Properties'",
	}
	var flag uint = 0

	call := conn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, flag)
	if call.Err != nil {
		fmt.Fprintln(os.Stderr, "Failed to become monitor:", call.Err)
		os.Exit(1)
	}

	client.Login("902662551119224852")
	c := make(chan *dbus.Message, 10)
	conn.Eavesdrop(c)
	fmt.Println("Listening on music status..")
	for msg := range c {
		if len(msg.Body) <= 1 {
			continue
		}
		bodyMap := msg.Body[1].(map[string]dbus.Variant)
		if bodyMap["Metadata"].Value() == nil {
			continue
		}
		metadata := bodyMap["Metadata"].Value().(map[string]dbus.Variant)
		artists := strings.Join(metadata["xesam:artist"].Value().([]string), ", ")
		client.SetActivity(client.Activity{
			Details:    metadata["xesam:title"].Value().(string),
			State:      "on " + metadata["xesam:album"].Value().(string) + " by " + artists,
			LargeImage: "music",
			LargeText:  "cmus",
			SmallImage: "playing",
			SmallText:  "Playing",
			/*Timestamps: &client.Timestamps{
				Start: &start,
			},*/
		})
	}

	<-done
}
