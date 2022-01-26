package main

type config struct {
	Blacklist []string `json:"blacklist"`
	Presence presenceConfig `json:"presence"`
	PlayerPresence map[string]presenceConfig `json:"playerPresence"`
	Vars []string `json:"vars"`
}

type presenceConfig struct {
	ShowTime bool `json:"showTime"`
	State string `json:"state"`
	Details string `json:"details"`
}

func (c *config) playerConfig(player string) presenceConfig {
	for name, conf := range c.PlayerPresence {
		if name == player {
			return conf
		}
	}

	return c.Presence
}
