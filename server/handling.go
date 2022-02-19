package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"cernobor.cz/oko-server/errs"
	"cernobor.cz/oko-server/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func internalError(gc *gin.Context, err error) {
	gc.String(http.StatusInternalServerError, "%v", err)
}

func (s *Server) setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// logging
	ginLogger := logrus.New()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	router.Use(func(gc *gin.Context) {
		path := gc.Request.URL.Path
		start := time.Now()
		gc.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1_000_000.0))
		statusCode := gc.Writer.Status()
		clientIP := gc.ClientIP()
		clientUserAgent := gc.Request.UserAgent()
		referer := gc.Request.Referer()
		dataLength := gc.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}
		entry := ginLogger.WithFields(logrus.Fields{
			"hostname":   hostname,
			"statusCode": statusCode,
			"latency":    latency,
			"clientIP":   clientIP,
			"method":     gc.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": dataLength,
			"userAgent":  clientUserAgent,
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
	})

	// utility/debug paths
	router.GET(URIPing, func(gc *gin.Context) {
		gc.Status(http.StatusNoContent)
	})
	router.GET(URIHardFail, func(gc *gin.Context) {
		gc.Status(http.StatusNotImplemented)
	})
	router.GET(URISoftFail, func(gc *gin.Context) {
		gc.JSON(http.StatusOK, map[string]string{"error": "artificial fail"})
	})
	router.POST(URIReinit, s.handlePOSTReset)

	// resources
	router.GET(URIMapPack, s.handleGETTilepack)

	// API
	router.POST(URIHandshake, s.handlePOSTHandshake)
	router.GET(URIData, s.handleGETData)
	router.POST(URIData, s.handlePOSTData)
	router.GET(URIDataPeople, s.handleGETDataPeople)
	router.GET(URIDataFeatures, s.handleGETDataFeatures)
	router.GET(URIDataFeaturesPhoto, s.handleGETDataFeaturesPhoto)

	// tileserver
	router.GET(URITileserver, gin.WrapH(s.tileserverSvSet.Handler()))

	return router
}

func (s *Server) handlePOSTReset(gc *gin.Context) {
	err := s.reinitDB()
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.Status(http.StatusOK)
}

func (s *Server) handleGETTilepack(gc *gin.Context) {
	gc.File(s.config.TilepackPath)
}

func (s *Server) handlePOSTHandshake(gc *gin.Context) {
	var hs models.HandshakeChallenge
	err := gc.ShouldBindJSON(&hs)
	if err != nil {
		gc.String(http.StatusBadRequest, fmt.Sprintf("malformed handshake challenge: %v", err))
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
	accept := gc.GetHeader("Accept")
	if accept == "application/json" {
		data, err := s.getDataOnly()
		if err != nil {
			internalError(gc, err)
			return
		}
		gc.JSON(http.StatusOK, data)
		return
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
		return
	}
	gc.String(http.StatusNotAcceptable, "%s is not acceptable", accept)
}

func (s *Server) handlePOSTData(gc *gin.Context) {
	switch gc.ContentType() {
	case "application/json":
		s.handlePOSTDataJSON(gc)
	case "multipart/form-data":
		s.handlePOSTDataMultipart(gc)
	default:
		gc.String(http.StatusBadRequest, "unsupported Content-Type")
	}
}

func (s *Server) handlePOSTDataJSON(gc *gin.Context) {
	var data models.Update
	err := gc.ShouldBindJSON(&data)
	if err != nil {
		gc.String(http.StatusBadRequest, fmt.Sprintf("malformed data: %v", err))
		return
	}

	if !isUniqueFeatureID(data.Create) {
		gc.String(http.StatusBadRequest, "created features do not have unique IDs")
		return
	}

	if data.CreatedPhotos != nil || data.AddPhotos != nil {
		gc.String(http.StatusBadRequest, "created_photos and/or add_photos present, but Content-Type is application/json")
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
		gc.String(http.StatusBadRequest, "malformed multipart/form-data content")
		return
	}

	dataStr, ok := form.Value["data"]
	if !ok {
		dataFile, ok := form.File["data"]
		if !ok {
			gc.String(http.StatusBadRequest, "value 'data' is missing from the content")
			return
		}
		if len(dataFile) != 1 {
			gc.String(http.StatusBadRequest, "value 'data' does not contain exactly 1 item")
			return
		}
		df, err := dataFile[0].Open()
		if err != nil {
			internalError(gc, fmt.Errorf("failed to open 'data' 'file': %w", err))
			return
		}
		dataBytes, err := ioutil.ReadAll(df)
		if err != nil {
			internalError(gc, fmt.Errorf("failed to open 'data' 'file': %w", err))
			return
		}
		dataStr = []string{string(dataBytes)}
	}
	if len(dataStr) != 1 {
		gc.String(http.StatusBadRequest, "value 'data' does not contain exactly 1 item")
		return
	}

	var data models.Update
	err = json.Unmarshal([]byte(dataStr[0]), &data)
	if err != nil {
		gc.String(http.StatusBadRequest, "malformed 'data' value: %v", err)
		return
	}

	if !isUniqueFeatureID(data.Create) {
		gc.String(http.StatusBadRequest, "created features do not have unique IDs")
		return
	}

	photos := make(map[string]models.Photo, len(form.File))
	for name, fh := range form.File {
		if len(fh) != 1 {
			gc.String(http.StatusBadRequest, "file item %s does not contain exactly 1 file", name)
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
			gc.String(http.StatusBadRequest, e.Error())
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
		gc.String(http.StatusBadRequest, "malformed feature ID")
		return
	}
	reqPhotoID, err := strconv.Atoi(gc.Param("photo"))
	if err != nil {
		gc.String(http.StatusBadRequest, "malformed photo ID")
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
