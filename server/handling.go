package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"cernobor.cz/oko-server/errs"
	"cernobor.cz/oko-server/models"
	"github.com/coreos/go-semver/semver"
	"github.com/gin-gonic/gin"
	"github.com/mssola/user_agent"
	"github.com/sirupsen/logrus"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func internalError(gc *gin.Context, err error) {
	gc.Error(err)
	gc.String(http.StatusInternalServerError, "%v", err)
}

func badRequest(gc *gin.Context, err error) {
	gc.Error(err)
	gc.String(http.StatusBadRequest, "%v", err)
}

func (s *Server) setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// logging
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	router.Use(
		func(gc *gin.Context) {
			path := gc.Request.URL.Path
			start := time.Now()
			gc.Next()
			stop := time.Since(start)
			latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1_000_000.0))
			statusCode := gc.Writer.Status()
			clientIP := gc.ClientIP()
			clientUserAgent := gc.Request.UserAgent()
			userId := gc.GetHeader(UserIDHeader)
			referer := gc.Request.Referer()
			dataLength := gc.Writer.Size()
			if dataLength < 0 {
				dataLength = 0
			}
			entry := s.log.WithFields(logrus.Fields{
				"hostname":   hostname,
				"statusCode": statusCode,
				"latency":    latency,
				"clientIP":   clientIP,
				"method":     gc.Request.Method,
				"path":       path,
				"referer":    referer,
				"dataLength": dataLength,
				"userAgent":  clientUserAgent,
				"userID":     userId,
			})

			if len(gc.Errors) > 0 {
				entry.Error(gc.Errors.ByType(gin.ErrorTypePrivate).String())
			} else {
				msg := fmt.Sprintf(
					"%s - %s [%s] \"%s %s\" %d %d \"%s\" \"%s\" (%dms)",
					clientIP, hostname, time.Now().Format(time.RFC3339), gc.Request.Method, path, statusCode, dataLength, referer, clientUserAgent, latency,
				)
				if statusCode >= 500 {
					entry.Error(msg)
				} else if statusCode >= 400 {
					entry.Warn(msg)
				} else if path == URIPing {
					entry.Debug(msg)
				} else {
					entry.Info(msg)
				}
			}
		},
		func(gc *gin.Context) {
			if !s.dbAvailable.Load() {
				gc.AbortWithError(http.StatusServiceUnavailable, fmt.Errorf("server database is not ready/available"))
				gc.Header("Retry-After", "60")
				return
			}
			gc.Next()
		},
		func(gc *gin.Context) {
			userIdStr := gc.GetHeader(UserIDHeader)
			userIdPresent := userIdStr != ""
			var userId int64
			var userIdErr error
			if userIdPresent {
				userId, userIdErr = strconv.ParseInt(userIdStr, 10, 0)
			}
			appVersion, appVersionErr := extractAppVersion(gc)
			if userIdErr != nil {
				gc.Error(fmt.Errorf("malformed %s: %w", UserIDHeader, userIdErr))
			} else if userIdPresent {
				gc.Set("uid", userId)
				var err error
				if appVersion == nil {
					err = withDbConn(s, func(conn *sqlite.Conn) error {
						err := sqlitex.Execute(conn, "update users set last_seen_time = ? where id = ?", &sqlitex.ExecOptions{
							Args: []interface{}{time.Now().Unix(), userId},
						})
						return err
					})
				} else {
					err = withDbConn(s, func(conn *sqlite.Conn) error {
						err := sqlitex.Execute(conn, "update users set app_version = ?, last_seen_time = ? where id = ?", &sqlitex.ExecOptions{
							Args: []interface{}{appVersion, time.Now().Unix(), userId},
						})
						return err
					})
				}
				if err != nil {
					gc.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to store user data into db: %w", err))
				}
			}
			if appVersion != nil {
				gc.Set("app-version", appVersion)
			}
			if appVersionErr != nil {
				gc.Error(appVersionErr)
				gc.Set("app-version-error", appVersionErr.Error())
			}
			gc.Next()
		},
	)

	// tileserver
	router.GET(URITileserver, gin.WrapH(s.tileserverSvSet.Handler()))

	/*** API ***/
	router.GET(URIPing, s.handleGETPing)
	router.POST(URIHandshake, s.handlePOSTHandshake)
	router.GET(URIData, s.handleGETData)
	router.POST(URIData, s.handlePOSTData)
	router.GET(URIDataPeople, s.handleGETDataPeople)
	router.GET(URIDataFeatures, s.handleGETDataFeatures)
	router.GET(URIDataFeaturesPhoto, s.handleGETDataFeaturesPhoto)
	router.GET(URIDataProposals, s.handleGETDataProposals)

	// resources
	router.GET(URIMapPack, s.handleGETTilepack)

	// utility/debug paths
	router.GET(URIBuildInfo, func(gc *gin.Context) {
		gc.JSON(http.StatusOK, models.BuildInfo{
			VersionHash: s.config.VersionHash,
			BuildTime:   s.config.BuildTime,
		})
	})
	router.GET(URIHardFail, func(gc *gin.Context) {
		gc.Status(http.StatusNotImplemented)
	})
	router.GET(URISoftFail, func(gc *gin.Context) {
		gc.JSON(http.StatusOK, map[string]string{"error": "artificial fail"})
	})
	router.GET(URIAppVersions, s.handleGETAppVersions)
	router.POST(URIAppVersions, s.handlePOSTAppVersions)
	router.GET(URIAppVersion, s.handleGETAppVersion)
	router.DELETE(URIAppVersion, s.handleDELETEAppVersion)
	router.POST(URIReinit, s.handlePOSTReset)
	router.GET(URIUsageInfo, s.handleGETUsageInfo)

	return router
}

func extractAppVersion(gc *gin.Context) (*semver.Version, error) {
	ua := user_agent.New(gc.Request.UserAgent())
	n, v := ua.Browser()
	if n != AppName {
		return nil, nil
	}

	version, err := semver.NewVersion(v)
	if err != nil {
		return nil, fmt.Errorf("malformed version in User-Agent header: %w", err)
	}
	return version, nil
}

func (s *Server) handleGETPing(gc *gin.Context) {
	versionRaw, exists := gc.Get("app-version")
	if !exists {
		versionErr := gc.GetString("app-version-error")
		if versionErr != "" {
			badRequest(gc, fmt.Errorf("malformed app version: %v", &versionErr))
		} else {
			badRequest(gc, fmt.Errorf("app version not specified"))
		}
		return
	}
	version, ok := versionRaw.(*semver.Version)
	if !ok {
		internalError(gc, fmt.Errorf("malformed app version extracted"))
		return
	}

	res, err := s.getLatestVersion(version)
	if err != nil {
		internalError(gc, err)
		return
	}

	if res == nil {
		gc.Status(http.StatusNoContent)
	} else {
		gc.JSON(http.StatusOK, res)
	}
}

func (s *Server) handleGETAppVersions(gc *gin.Context) {
	versions, err := s.getAppVersions()
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.JSON(http.StatusOK, versions)
}

func (s *Server) handlePOSTAppVersions(gc *gin.Context) {
	var versionInfo models.AppVersionInfo
	err := gc.ShouldBindJSON(&versionInfo)
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed version info: %w", err))
		return
	}

	err = s.putAppVersion(&versionInfo)
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.Status(http.StatusNoContent)
}

func (s *Server) handleGETAppVersion(gc *gin.Context) {
	version := gc.Param("version")
	if version == "" {
		badRequest(gc, fmt.Errorf("version not provided"))
		return
	}

	ver, err := s.getAppVersion(version)
	if err != nil {
		internalError(gc, err)
		return
	}
	if ver == nil {
		gc.Status(http.StatusNotFound)
		return
	}
	gc.JSON(http.StatusOK, ver)
}

func (s *Server) handleDELETEAppVersion(gc *gin.Context) {
	version := gc.Param("version")
	if version == "" {
		badRequest(gc, fmt.Errorf("version not provided"))
		return
	}

	deleted, err := s.deleteAppVersion(version)
	if err != nil {
		internalError(gc, err)
		return
	}
	if !deleted {
		gc.Status(http.StatusNotFound)
		return
	}
	gc.Status(http.StatusNoContent)
}

func (s *Server) handlePOSTReset(gc *gin.Context) {
	err := s.reinitDb()
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.Status(http.StatusNoContent)
}

func (s *Server) handleGETTilepack(gc *gin.Context) {
	gc.File(s.config.TilepackPath)
}

func (s *Server) handlePOSTHandshake(gc *gin.Context) {
	var hs models.HandshakeChallenge
	err := gc.ShouldBindJSON(&hs)
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed handshake challenge: %w", err))
		return
	}

	id, err := s.handshake(hs)
	if err != nil {
		if errors.Is(err, errs.ErrUserAlreadyExists) {
			gc.Status(http.StatusConflict)
		} else if errors.Is(err, errs.ErrUserNotExists) {
			gc.Status(http.StatusNotFound)
		} else if errors.Is(err, errs.ErrAttemptedSystemUser) {
			gc.Status(http.StatusForbidden)
		} else {
			internalError(gc, err)
		}
		return
	}

	gc.JSON(http.StatusOK, models.HandshakeResponse{
		ID:   id,
		Name: hs.Name,
		MapInfo: models.MapInfo{
			MapPackPath:      URIMapPack,
			MapPackSize:      s.mapPackSize,
			TilePathTemplate: URITileTemplate,
			MinZoom:          s.config.MinZoom,
			DefaultCenter:    s.config.DefaultCenter,
		},
	})
}

func (s *Server) handleGETData(gc *gin.Context) {
	uidRaw, uidExists := gc.Get("uid")
	uid, _ := uidRaw.(int64)
	accept := gc.GetHeader("Accept")
	if accept == "application/json" {
		data, err := s.getDataOnly()
		if err != nil {
			internalError(gc, err)
			return
		}
		gc.JSON(http.StatusOK, data)
	} else if accept == "application/zip" {
		file, err := s.getDataWithPhotos()
		defer func() {
			file.Close()
			os.Remove(file.Name())
		}()
		if err != nil {
			internalError(gc, err)
			return
		}
		fi, err := file.Stat()
		if err != nil {
			internalError(gc, err)
			return
		}
		size := fi.Size()
		_, err = file.Seek(0, 0)
		if err != nil {
			internalError(gc, err)
			return
		}
		gc.DataFromReader(http.StatusOK, size, "application/zip", file, nil)
	} else {
		gc.String(http.StatusNotAcceptable, "%s is not acceptable", accept)
		return
	}
	if uidExists {
		err := withDbConn(s, func(conn *sqlite.Conn) error {
			return sqlitex.Execute(conn, "update users set last_download_time = ? where id = ?", &sqlitex.ExecOptions{
				Args: []interface{}{time.Now().Unix(), uid},
			})
		})
		if err != nil {
			gc.Error(fmt.Errorf("failed to store last download time: %w", err))
		}
	}
}

func (s *Server) handlePOSTData(gc *gin.Context) {
	uidRaw, uidExists := gc.Get("uid")
	uid, _ := uidRaw.(int64)
	switch gc.ContentType() {
	case "application/json":
		s.handlePOSTDataJSON(gc)
	case "multipart/form-data":
		s.handlePOSTDataMultipart(gc)
	default:
		badRequest(gc, fmt.Errorf("unsupported Content-Type"))
		return
	}
	if uidExists {
		err := withDbConn(s, func(conn *sqlite.Conn) error {
			return sqlitex.Execute(conn, "update users set last_upload_time = ? where id = ?", &sqlitex.ExecOptions{
				Args: []interface{}{time.Now().Unix(), uid},
			})
		})
		if err != nil {
			gc.Error(fmt.Errorf("failed to store last upload time: %w", err))
		}
	}
}

func (s *Server) handlePOSTDataJSON(gc *gin.Context) {
	var data models.Update
	err := gc.ShouldBindJSON(&data)
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed data: %w", err))
		return
	}

	if !isUniqueFeatureID(data.Create) {
		badRequest(gc, fmt.Errorf("created features do not have unique IDs"))
		return
	}

	if data.CreatedPhotos != nil || data.AddPhotos != nil {
		badRequest(gc, fmt.Errorf("created_photos and/or add_photos present, but Content-Type is application/json"))
		return
	}

	err = s.update(data, nil)
	if err != nil {
		internalError(gc, fmt.Errorf("failed to update data: %w", err))
		return
	}
	gc.Status(http.StatusNoContent)
}

func (s *Server) handlePOSTDataMultipart(gc *gin.Context) {
	form, err := gc.MultipartForm()
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed multipart/form-data content"))
		return
	}

	dataStr, ok := form.Value["data"]
	if !ok {
		dataFile, ok := form.File["data"]
		if !ok {
			badRequest(gc, fmt.Errorf("value 'data' is missing from the content"))
			return
		}
		if len(dataFile) != 1 {
			badRequest(gc, fmt.Errorf("value 'data' does not contain exactly 1 item"))
			return
		}
		df, err := dataFile[0].Open()
		if err != nil {
			internalError(gc, fmt.Errorf("failed to open 'data' 'file': %w", err))
			return
		}
		dataBytes, err := io.ReadAll(df)
		if err != nil {
			internalError(gc, fmt.Errorf("failed to open 'data' 'file': %w", err))
			return
		}
		dataStr = []string{string(dataBytes)}
	}
	if len(dataStr) != 1 {
		badRequest(gc, fmt.Errorf("value 'data' does not contain exactly 1 item"))
		return
	}

	var data models.Update
	err = json.Unmarshal([]byte(dataStr[0]), &data)
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed 'data' value: %w", err))
		return
	}

	if !isUniqueFeatureID(data.Create) {
		badRequest(gc, fmt.Errorf("created features do not have unique IDs"))
		return
	}

	photos := make(map[string]models.Photo, len(form.File))
	for name, fh := range form.File {
		if len(fh) != 1 {
			badRequest(gc, fmt.Errorf("file item %s does not contain exactly 1 file", name))
			return
		}
		var photo models.Photo
		f := fh[0]
		photo.ContentType = f.Header.Get("Content-Type")
		photo.Size = f.Size
		photo.File, err = f.Open()
		if err != nil {
			internalError(gc, fmt.Errorf("failed to open provided photo file: %w", err))
		}
		defer photo.File.Close()
		photos[name] = photo
	}
	err = s.update(data, photos)
	if err != nil {
		var e *errs.ErrUnsupportedContentType
		if errors.As(err, &e) {
			badRequest(gc, e)
			return
		}
		internalError(gc, fmt.Errorf("failed to update data: %w", err))
		return
	}

	gc.Status(http.StatusNoContent)
}

func (s *Server) handleGETDataPeople(gc *gin.Context) {
	people, err := s.getPeople(nil)
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.JSON(http.StatusOK, people)
}

func (s *Server) handleGETDataFeatures(gc *gin.Context) {
	pois, err := s.getFeatures(nil)
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.JSON(http.StatusOK, pois)
}

func (s *Server) handleGETDataFeaturesPhoto(gc *gin.Context) {
	reqFeatureID, err := strconv.Atoi(gc.Param("feature"))
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed feature ID"))
		return
	}
	reqPhotoID, err := strconv.Atoi(gc.Param("photo"))
	if err != nil {
		badRequest(gc, fmt.Errorf("malformed photo ID"))
		return
	}

	photoBytes, contentType, err := s.getPhoto(models.FeatureID(reqFeatureID), models.FeaturePhotoID(reqPhotoID))
	if err != nil {
		if errors.Is(err, errs.ErrPhotoNotExists) {
			gc.String(http.StatusNotFound, "%v", err)
		} else {
			internalError(gc, fmt.Errorf("failed to retrieve photo: %w", err))
		}
		return
	}

	gc.Data(http.StatusOK, contentType, photoBytes)
}

func (s *Server) handleGETDataProposals(gc *gin.Context) {
	proposals, err := s.getProposals(nil)
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.JSON(http.StatusOK, proposals)
}

func (s *Server) handleGETUsageInfo(gc *gin.Context) {
	usage, err := s.getUsageInfo(nil)
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.JSON(http.StatusOK, usage)
}
