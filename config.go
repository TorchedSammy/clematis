package main

type config struct {
	Blacklist []string `json:"blacklist"`
	Presence presenceConfig `json:"presence"`
	PlayerPresence map[string]presenceConfig `json:"playerPresence"`
	Vars []string `json:"vars"`
	LogLevel string `json:"logLevel"`
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
