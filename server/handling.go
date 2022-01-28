package server

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"cernobor.cz/oko-server/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func internalError(gc *gin.Context, err error) {
	gc.String(http.StatusInternalServerError, "%v", err)
}

func (s *Server) setupRouter() *gin.Engine {
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

	// resources
	router.GET(URITilepack, s.handleGETTilepack)

	// API
	router.POST(URIHandshake, s.handlePOSTHandshake)
	router.GET(URIData, s.handleGETData)
	router.POST(URIData, s.handlePOSTData)
	router.GET(URIDataPeople, s.handleGETDataPeople)
	router.GET(URIDataFeatures, s.handleGETDataFeatures)

	// tileserver
	router.GET(URITileserver, gin.WrapH(s.tileserverSvSet.Handler()))

	return router
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
		if errors.Is(err, ErrUserAlreadyExists) {
			gc.Status(http.StatusConflict)
		} else if errors.Is(err, ErrUserNotExists) {
			gc.Status(http.StatusNotFound)
		} else if errors.Is(err, ErrAttemptedSystemUser) {
			gc.Status(http.StatusForbidden)
		} else {
			internalError(gc, err)
		}
		return
	}

	gc.JSON(http.StatusOK, models.HandshakeResponse{
		ID:           id,
		Name:         hs.Name,
		TilePackPath: URITilepack,
	})
}

func (s *Server) handleGETData(gc *gin.Context) {
	data, err := s.getData()
	if err != nil {
		internalError(gc, err)
		return
	}
	gc.JSON(http.StatusOK, data)
}

func (s *Server) handlePOSTData(gc *gin.Context) {
	var data models.Update
	err := gc.ShouldBindJSON(&data)
	if err != nil {
		gc.String(http.StatusBadRequest, fmt.Sprintf("malformed data: %v", err))
		return
	}

	err = s.update(data)
	if err != nil {
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
