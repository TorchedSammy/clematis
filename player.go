package main

import (
	"errors"
	"strings"

	"github.com/Pauloo27/go-mpris"
	"github.com/godbus/dbus/v5"
)

var (
	errAllBlacklisted = errors.New("All players are blacklisted")
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

	playerName := ""
	// get first player name, unless it's in the blacklist
	for _, propName := range names {
		namePieces := strings.Split(propName, ".")
		name := namePieces[len(namePieces) - 1]

		if !contains(conf.Blacklist, name) {
			return propName, nil
		}
	}

	if playerName == "" {
		return "", errAllBlacklisted
	}

	return "", nil // unreachable
}
