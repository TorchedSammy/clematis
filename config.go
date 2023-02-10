package main

type config struct {
	Blacklist []string `json:"blacklist"`
	Whitelist []string `json:"whitelist"`
	UseIdentifiers bool `json:"useIdentifiers"`
	Presence presenceConfig `json:"presence"`
	PlayerPresence map[string]presenceConfig `json:"playerPresence"`
	Vars []string `json:"vars"`
	LogLevel string `json:"logLevel"`
	ShowAlbumArt bool `json:"showAlbumArt"`
	ArtFetchMethod string `json:"artFetchMethod"`
}

type presenceConfig struct {
	Details string `json:"details"`
	State string `json:"state"`
	ShowTime bool `json:"showTime"`
}

func (c *config) playerConfig(player string) presenceConfig {
	for name, conf := range c.PlayerPresence {
		if name == player {
			return conf
		}
	}

	return c.Presence
}
