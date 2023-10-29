package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"net/url"

	"github.com/godbus/dbus/v5"
)

var tokenRegex = regexp.MustCompile(`mfa\.[\w-]{84}`)
var receivedAssets []Asset

type discordFetcher struct{}

func (discordFetcher) getAlbumArt(artist, album, title string, mdata map[string]dbus.Variant) string {
	artFile := ""
	if artUrl, ok := mdata["mpris:artUrl"].Value().(string); ok {
		artFile, _ = url.PathUnescape(artUrl)
		// remove file:// from the beginning
		artFile = artFile[7:]
	}

	albumAsset, err := checkForAsset(album)
	fmt.Println(err, albumAsset)
	if err != nil {
		fmt.Println("Uploading " + artFile + " to discord")
		albumAsset, err = uploadAsset(artFile, album)
		fmt.Println(err)
	}

	return albumAsset
}

// function to get token from local discord db
func getDiscordToken() string {
	discordDir := os.Getenv("HOME") + "/.config/discord"
	dbDir := discordDir + "/Local Storage/leveldb/"
	// get files in dbDir
	files, err := os.ReadDir(dbDir)
	dbs := []fs.DirEntry{}
	if err != nil {
		fmt.Println(err)
		return ""
	}
	// add file to dbs if it ends with .ldb
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".ldb") {
			dbs = append(dbs, file)
		}
	}

	// sort by modification time
	sort.Slice(files, func(i, j int) bool {
		firstFileInfo, _ := files[i].Info()
		secondFileInfo, _ := files[j].Info()
		return firstFileInfo.ModTime().After(secondFileInfo.ModTime())
	})

	// go through all leveldbs to find the one with the token
	for _, dbFile := range dbs {
		dbContents, err := os.ReadFile(dbDir + dbFile.Name())
		if err != nil {
			fmt.Println(err)
			return ""
		}

		// return single regex match
		token := tokenRegex.FindString(string(dbContents))
		if token != "" {
			return token
		}
	}

	return ""
}

func getAssets() []Asset {
	// make get request to get assets
	if len(receivedAssets) != 0 {
		return receivedAssets
	}
	resp, err := http.Get("https://discord.com/api/v9/oauth2/applications/902662551119224852/assets")
	if err != nil {
		fmt.Println(err)
		return []Asset{}
	}

	var assets []Asset
	json.NewDecoder(resp.Body).Decode(&assets)

	receivedAssets = assets
	return receivedAssets
}

// upload asset to discord
// assetName will be the album name, or song name if no album
// "music" is returned as the assetName when an error occurs since that's the name of just a music icon
func uploadAsset(fileName, assetName string) (string, error) {
	image, err := os.ReadFile(fileName)
	if err != nil {
		return "music", err
	}

	base64Encoding := ""

	// determine the content type of the image file
	imageType := http.DetectContentType(image)

	// album art is only going to be jpg or png, right?
	// if not i hate you
	switch imageType {
	case "image/jpeg":
		base64Encoding += "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding += "data:image/png;base64,"
	}

	base64Encoding += base64.StdEncoding.EncodeToString(image)

	// turn assetName into a hex encoded string
	assetNameEncoded := hex.EncodeToString([]byte(assetName))

	asset := AssetToUpload{
		Type: "1",
		Name: assetNameEncoded,
		Image: base64Encoding,
	}

	// make post request to upload asset, using token in Authentication header
	jsonBytes, err := json.Marshal(asset)
	req, _ := http.NewRequest("POST", "https://discord.com/api/v9/oauth2/applications/902662551119224852/assets", bytes.NewBuffer(jsonBytes))

	req.Header.Set("Authorization", getDiscordToken())
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "music", err
	}
	var receivedAssetInfo Asset
	json.NewDecoder(resp.Body).Decode(&receivedAssetInfo)
	resp.Body.Close()
	fmt.Println(receivedAssetInfo)

	receivedAssets = append(receivedAssets, receivedAssetInfo)

	return assetName, nil
}

func checkForAsset(albumName string) (string, error) {
	// get assets from discord
	assets := getAssets()
	
	albumNameEncoded := hex.EncodeToString([]byte(albumName))

	// go through assets to see if one matches fileName
	for _, asset := range assets {
		if asset.Name == albumNameEncoded {
			fmt.Println("found asset")
			return asset.Name, nil
		}
	}

	return "", fmt.Errorf("No asset found for %s", albumName)
}
