package models

import (
	"io"
	"time"

	"github.com/coreos/go-semver/semver"
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

type UserInfo struct {
	User
	AppVersion       *semver.Version `json:"app_version"`
	LastSeenTime     *time.Time      `json:"last_seen_time"`
	LastUploadTime   *time.Time      `json:"last_upload_time"`
	LastDownloadTime *time.Time      `json:"last_download_time"`
}

type Feature struct {
	// ID is an ID of the feature.
	// When the feature is submitted by a client for creation (i.e. in Update.Create) it is considered a 'local' ID which must be unique across all submitted features.
	ID         FeatureID              `json:"id"`
	OwnerID    UserID                 `json:"owner_id"`
	Name       string                 `json:"name"`
	Deadline   *time.Time             `json:"deadline,omitempty"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   geojson.Geometry       `json:"geometry"`
	// PhotoIDs contains a list IDs of photos associated with the feature.
	// When the feature is retrieved from the server, the IDs can be used to retrieve the photos.
	// When the feature is submitted by a client (created or updated, i.e. in Update.Create or Update.Update), this list is ignored (as photos are managed through Update.CreatePhotos and Update.DeletePhotos fields).
	PhotoIDs []FeaturePhotoID `json:"photo_ids"`
}

type BuildInfo struct {
	VersionHash string     `json:"version_hash"`
	BuildTime   *time.Time `json:"build_time"`
}

// transport objects

type AppVersionInfo struct {
	Version semver.Version `json:"version"`
	Address string         `json:"address"`
}

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
	Proposals     []Proposal             `json:"proposals"`
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
	Users         []User                   `json:"users"`
	Features      []Feature                `json:"features"`
	PhotoMetadata map[string]PhotoMetadata `json:"photo_metadata,omitempty"`
}

type Photo struct {
	ContentType string
	File        io.ReadCloser
	Size        int64
}

type PhotoMetadata struct {
	ContentType          string         `json:"content_type"`
	ThumbnailContentType string         `json:"thumbnail_content_type"`
	Size                 int64          `json:"size"`
	ID                   FeaturePhotoID `json:"id"`
	ThumbnailFilename    string         `json:"thumbnail_filename"`
}

type Proposal struct {
	OwnerID     UserID `json:"owner_id"`
	Description string `json:"description"`
	How         string `json:"how"`
}
