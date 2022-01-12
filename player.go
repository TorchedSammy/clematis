package main

import (
	"errors"

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
		// get identity of each player
		player := mpris.New(conn, propName)
		identity, err := player.GetIdentity()
		if err != nil {
			panic(err)
		}

		if !contains(conf.Blacklist, identity) {
			return propName, nil
		}
	}

	if playerName == "" {
		return "", errAllBlacklisted
	}

	return "", nil // unreachable
}
