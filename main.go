package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Pauloo27/go-mpris"
	"github.com/godbus/dbus/v5"
	"github.com/hugolgst/rich-go/client"
	"github.com/spf13/pflag"
	"github.com/IchBinLeoon/slogx"
)

var version = "v0.3.0"
var pbStat string
var conf = config{
	Presence: presenceConfig{
		ShowTime: true,
		Details: "{title}",
		State: "{artist} {album}",
	},
	LogLevel: "info",
	ShowAlbumArt: true,
	ArtFetchMethod: "spotify",
}

var logger = slogx.NewLogger("Clematis")
var fetcher artFetcher

func main() {
	logger.SetFormat("${time} ${level} > ${message}")

	confDir, err := os.UserConfigDir()
	if err != nil {
		logger.Fatal("Error getting config dir: ", err)
	}
	defaultConfPath := filepath.Join(confDir, "Clematis", "config.json")

	helpflag := pflag.BoolP("help", "h", false, "Show this help message")
	versionflag := pflag.BoolP("version", "v", false, "Show version")

	var confPath string
	pflag.StringVarP(&confPath, "config", "c", defaultConfPath, "Path to config file")

	pflag.Parse()

	if *helpflag {
		pflag.PrintDefaults()
		os.Exit(0)
	}

	if *versionflag {
		fmt.Println("Clematis", version)
		os.Exit(0)
	}

	confFile, _ := os.ReadFile(confPath)
	if err != nil {
		logger.Error("Error reading config file: ", err)
	}
	json.Unmarshal(confFile, &conf)
	switch conf.ArtFetchMethod {
		case "spotify": fetcher = spotifyFetcher{}
		case "discord": fetcher = discordFetcher{}
		default: panic(fmt.Sprintf("Invalid album art fetcher %s. The valid options are: spotify, discord", conf.ArtFetchMethod))
	}

	logger.SetLevel(slogx.ParseLevel(conf.LogLevel))

	// watcher conn is for eavesdropping messages and is a monitor
	// infoConn is for getting other data at any time
	// we can't Call on the watcher conn (error of EOF), so we use the separate infoConn
	watcherConn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	infoConn, _ := dbus.ConnectSessionBus()

	playerName, err := getPlayerName(infoConn, conf)
	if err == errNoPlayers {
		logger.Fatal("No MPRIS players found.")
	} else if err == errAllExcluded {
		logger.Fatal("Could not find any player that is not excluded (blacklisted or non-whitelisted).")
	}

	player := mpris.New(infoConn, playerName)
	playerIdentity, _ := player.GetIdentity()
	logger.Info("Getting information from ", playerIdentity)

	var rules = []string{
		"type='signal',member='PropertiesChanged',path='/org/mpris/MediaPlayer2',interface='org.freedesktop.DBus.Properties'",
		"type='signal',member='Seeked',path='/org/mpris/MediaPlayer2',interface='org.mpris.MediaPlayer2.Player'",
		"member='NameLost'",
		"member='NameAcquired'",
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
		logger.Error(err)
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
		logger.Debug(msg.Body)
		// only check if seek is from our main player
		if msgMember == "Seeked" && msg.Headers[dbus.FieldSender].Value().(string) == playerConnName {
			// if the "seek" is at the beginning of the song
			// we dont want to update the presence
			// because a seek with a value of 0 is sent when a song starts,
			// this is to prevent a triple update of the presence
			// playing status will be sent when the song restarts
			if msg.Body[0].(int64) != 0 {
				logger.Debug("Player seeked")
				elapsed, _ := player.GetPosition()
				setPresence(*mdata, time.Now().Add(-time.Duration(elapsed) * time.Second), player)
			}		
		} else if msgMember == "NameLost" {
			if msg.Body[0] == playerName {
				logger.Info("Main player disconnected")
				// if there was another player connected to dbus
				playerName, err = getPlayerName(infoConn, conf)
				if err == nil {
					player = mpris.New(infoConn, playerName)
					playerIdentity, _ = player.GetIdentity()
					logger.Info("Switched to", playerIdentity)
					*mdata, elapsed, pbStat, playerConnName = getInitialData(player, infoConn, playerName)
					setPresence(*mdata, time.Now().Add(-time.Duration(elapsed) * time.Microsecond), player)
				} else {
					logger.Info("Logging out")
					client.Logout()
					player = nil
					continue
				}
			}
		} else if msgMember == "NameAcquired" {
			if player == nil {
				playerName, err = getPlayerName(infoConn, conf)
				if err != nil {
					continue // skip if the player is blacklisted
				}

				err = client.Login("902662551119224852")
				if err != nil {
					logger.Error(err)
					os.Exit(1)
				}

				player = mpris.New(infoConn, playerName)
				playerIdentity, _ = player.GetIdentity()
				logger.Info("Switched to", playerIdentity)
				*mdata, elapsed, pbStat, playerConnName = getInitialData(player, infoConn, playerName)
				setPresence(*mdata, time.Now().Add(-time.Duration(elapsed) * time.Microsecond), player)
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
	albumName := ""
	if abm, ok := metadata["xesam:album"].Value().(string); ok {
		albumName = abm
		album = "on " + albumName
	}
	if pbStat != "Playing" {
		startstamp, endstamp = nil, nil
	}

	artistsStr := ""
	if artistsArr, ok := metadata["xesam:artist"].Value().([]string); ok {
		artistsStr = "by " + strings.Join(artistsArr, ", ")
	}

	url := fetcher.getAlbumArt(artistsStr, albumName, metadata)

	args := []string{
		"{artist}", artistsStr,
		"{title}", title,
		"{album}", album,
	}
	for _, confVar := range conf.Vars {
		val := metadata[confVar].Value()
		if val != nil {
			if s, ok := val.(string); ok {
				args = append(args, "{" + confVar + "}", s)
			}
		}
	}
	replacer := strings.NewReplacer(args...)
	p := conf.playerConfig(playerIdentity)

	client.SetActivity(client.Activity{
		Details: replacer.Replace(p.Details),
		State: replacer.Replace(p.State),
		LargeImage: url,
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
