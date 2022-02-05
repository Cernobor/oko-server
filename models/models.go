package models

import (
	"io"

	geojson "github.com/paulmach/go.geojson"
)

// core objects

type UserID int64
type FeatureID int64
type FeaturePhotoID int64

type User struct {
	ID   UserID `json:"id"`
	Name string `json:"name"`
}

type Feature struct {
	// ID is an ID of the feature.
	// When the feature is submitted by a client for creation (i.e. in Update.Create) it is considered a 'local' ID which must be unique across all submitted features.
	ID         FeatureID              `json:"id"`
	OwnerID    UserID                 `json:"owner_id"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   geojson.Geometry       `json:"geometry"`
	// PhotoIDs contains a list IDs of photos associated with the feature.
	// When the feature is retrieved from the server, the IDs can be used to retrieve the photos.
	// When the feature is submitted by a client (created or updated, i.e. in Update.Create or Update.Update), this list is ignored (as photos are managed through Update.CreatePhotos and Update.DeletePhotos fields).
	PhotoIDs []FeaturePhotoID `json:"photo_ids"`
}

// transport objects

type Coords struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type MapInfo struct {
	MapPackPath      string `json:"map_pack_path"`
	MapPackSize      int64  `json:"map_pack_size"`
	TilePathTemplate string `json:"tile_path_template"`
	MinZoom          int    `json:"min_zoom"`
	DefaultCenter    Coords `json:"default_center"`
}

type Update struct {
	Create        []Feature              `json:"create"`
	CreatedPhotos map[FeatureID][]string `json:"created_photos"`
	AddPhotos     map[FeatureID][]string `json:"add_photos"`
	Update        []Feature              `json:"update"`
	Delete        []FeatureID            `json:"delete"`
	DeletePhotos  []FeaturePhotoID       `json:"delete_photos"`
}

type HandshakeChallenge struct {
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
}

type HandshakeResponse struct {
	ID      UserID  `json:"id"`
	Name    string  `json:"name"`
	MapInfo MapInfo `json:"map_info"`
}

type Data struct {
	Users    []User    `json:"users"`
	Features []Feature `json:"features"`
}

type Photo struct {
	ContentType string
	File        io.ReadCloser
	Size        int64
}
