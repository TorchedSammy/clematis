package main

import (
	"errors"
	"strings"

	"github.com/Pauloo27/go-mpris"
	"github.com/godbus/dbus/v5"
)

var (
	errAllExcluded = errors.New("All players are excluded (blacklisted or non-whitelisted)")
	errNoPlayers = errors.New("No players found")
)

func getPlayerName(conn *dbus.Conn, conf config) (string, error) {
	names, err := mpris.List(conn)
	if err != nil {
		panic(err)
	}

	if len(names) == 0 {
		return "", errNoPlayers
	}

	// get first player name, unless it's excluded
	for _, propName := range names {
		// get identity of each player
		identity := strings.TrimPrefix(propName, "org.mpris.MediaPlayer2.")

		isBlacklisted := contains(conf.Blacklist, identity)
		isWhitelisted := contains(conf.Whitelist, identity)
		whitelistDisabled := conf.Whitelist == nil || len(conf.Whitelist) == 0

		if !isBlacklisted && (whitelistDisabled || isWhitelisted) {
			return propName, nil
		}
	}

	return "", errAllExcluded
}
