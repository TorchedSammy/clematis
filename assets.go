package main

type Asset struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type AssetToUpload struct {
	Name string `json:"name"`
	Type string `json:"type"` // always 1
	Image string `json:"image"` // base64 encoded
}

