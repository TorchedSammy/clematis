# Clematis
> Discord rich presence for MPRIS music players.

Clematis provides a rich presence for [MPRIS-supported](#supported-clients) music
players. It will take information from the 1st music player and show it as a rich
presence.

> ⚠️ Clematis is currently a work in progress! It can error out at some times.

![](https://safe.kashima.moe/fgt0us64yvqu.png)

# Planned features
- Custom rich presence fields

# Installation
```
go install github.com/TorchedSammy/clematis
```

or by manually compiling:  
```sh
git clone https://github.com/TorchedSammy/clematis
cd Clematis
go get -d
go build
```

# Usage
Just run the `clematis` binary! Playing songs will automatically be made into the rich
presence.

# Configuration
Clematis can be configured using a JSON file. On Linux, the path is `~/.config/Clematis/config.json`.
`~/.config` can be changed to any dir with the `$XDG_CONFIG_HOME` env variable.  

It does not create a default config file, but for reference these are the configurable fields:  
```json5
{
	"vars": [""] // define a list of vars for the presence from the metadata (reference from `playerctl metadata`)
	"blacklist": [""], // list of blacklisted clients
	"presence": {
		"details": "{title}" // the top text in the presence
		"state": "{artist} {album}", // the bottom text in the presence
	},
	"playerPresence": {
		// define unique presence data per player
		// name used is what shows up on clematis startup
		"cmus": {
			// the fields here are the same as `presence`
		}
	}
}
```

The already provided metadata vars are:  
+ `title` - Name of the track
+ `artist` - Name of artist(s), joined by comma and prefixed with `by`
+ `album` - Name of album, prefixed with `on`
Vars can be used in the presence fields when they are surrounded by `{}`, like `{artist}`

# Supported Clients
Any music player that supports [MPRIS](https://specifications.freedesktop.org/mpris-spec/)
will work. This includes:
- cmus
- Chrome/Chromium
- VLC
- ncspot
- [mpv](https://github.com/hoyon/mpv-mpris)
- [mpd](https://wiki.archlinux.org/title/Music_Player_Daemon/Tips_and_tricks#MPRIS_support)

Some others are listed at the [Arch Linux wiki.](https://wiki.archlinux.org/title/MPRIS#Supported_clients)

# License
Clematis is licensed under the MIT license.  
[Read here](LICENSE) for more.

