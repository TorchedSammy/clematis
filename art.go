package main

// credit to https://github.com/lacymorrow/album-art
// thanks for the token too :)

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	// "sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	art_endpoint = "https://api.spotify.com/v1"
	auth_endpoint = "https://accounts.spotify.com/api/token"
	art_api_id = "3f974573800a4ff5b325de9795b8e603"
	art_api_secret = "ff188d2860ff44baa57acc79c121a3b9"
	art_api_auth = art_api_id + ":" + art_api_secret
)

type spotifyAccess struct{
	AccessToken string `json:"access_token"`
}

type spotifySeach struct{
	Albums spotifyAlbums
}

type spotifyAlbums struct{
	Items []spotifyItem
}

type spotifyItem struct{
	Images []spotifyArt
}

type spotifyArt struct {
	Width int
	Height int
	URL string
}

func getAlbumArt(artist, album string, mdata map[string]dbus.Variant) string {
	artUrl, _ := url.Parse(fmt.Sprintf("%s/search?q=%s&type=album&limit=1", art_endpoint, url.QueryEscape(artist + " " + album)))
	authUrl, _ := url.Parse(auth_endpoint)

	req, err := http.NewRequest("POST", authUrl.String(), strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		// TODO: dont panic
		panic(err)
	}
	req.Header.Add("Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte(art_api_auth)))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	spot := &spotifyAccess{}
	if err := json.Unmarshal(body, &spot); err != nil {
		panic(err)
	}

	req, err = http.NewRequest("GET", artUrl.String(), strings.NewReader(""))
	req.Header.Add("Authorization", "Bearer " + spot.AccessToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	body, err = io.ReadAll(resp.Body)
	logger.Debug(string(body))

	spotifyData := &spotifySeach{}
	if err := json.Unmarshal(body, &spotifyData); err != nil {
		panic(err)
	}

	images := spotifyData.Albums.Items[0].Images
	// may or may not be needed (will see)
	/*
	sort.Slice(images, func(i, j int) bool {
		return images[i].Width > images[j].Width
	})
	*/

	return images[0].URL
}
