package models

import geojson "github.com/paulmach/go.geojson"

// core objects

type UserID int
type FeatureID int

type User struct {
	ID   UserID `json:"id"`
	Name string `json:"name"`
}

type Feature struct {
	ID          FeatureID        `json:"id"`
	OwnerID     *UserID          `json:"owner_id"`
	Name        string           `json:"name"`
	Description *string          `json:"description"`
	Category    *string          `json:"category"`
	Geometry    geojson.Geometry `json:"geometry"`
}

// transport objects

type Update struct {
	Create []Feature   `json:"create"`
	Update []Feature   `json:"update"`
	Delete []FeatureID `json:"delete"`
}

type HandshakeChallenge struct {
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
}

type HandshakeResponse struct {
	ID           UserID `json:"id"`
	Name         string `json:"name"`
	TilePackPath string `json:"tile_pack_path"`
}

type Data struct {
	Users    []User    `json:"users"`
	Features []Feature `json:"features"`
}
