# Clematis
> Discord rich presence for MPRIS music players.

Clematis provides a rich presence for [MPRIS-supported](#supported-clients) music
players. It will take information from the 1st music player and show it as a rich
presence.

![](https://modeus.is-inside.me/0QEZeYnX.png)

# Installation
```
go install github.com/TorchedSammy/Clematis
```

or by manual compile:  
```sh
git clone https://github.com/TorchedSammy/Clematis
cd Clematis
go get -d
go build
```

# Usage
Just run the `Clematis` binary! Playing songs will automatically be made into the rich
presence.

# Supported Clients
Any music player that supports [MPRIS](https://specifications.freedesktop.org/mpris-spec/)
will work. This includes:
- cmus
- VLC
- ncspot
- [mpv](https://github.com/hoyon/mpv-mpris)
- [mpd](https://wiki.archlinux.org/title/Music_Player_Daemon/Tips_and_tricks#MPRIS_support)
Some others are listed at the [Arch Linux wiki.](https://wiki.archlinux.org/title/MPRIS#Supported_clients)

# License
Clematis is licensed under the MIT license.  
[Read here](LICENSE) for more.

