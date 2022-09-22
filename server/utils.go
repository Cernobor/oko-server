package server

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"

	"golang.org/x/image/draw"

	"cernobor.cz/oko-server/models"
)

func ptr[T any](x T) *T {
	return &x
}

var contentTypes map[string]struct{} = map[string]struct{}{"image/jpeg": {}, "image/png": {}}

func checkImageContentType(contentType string) bool {
	_, ok := contentTypes[contentType]
	return ok
}

func isUniqueFeatureID(features []models.Feature) bool {
	ids := make(map[models.FeatureID]struct{}, len(features))
	for _, f := range features {
		if _, pres := ids[f.ID]; pres {
			return false
		}
	}
	return true
}

func resizePhoto(maxX, maxY, quality int, data io.Reader) ([]byte, error) {
	src, _, err := image.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	if maxX == 0 && maxY == 0 {
		output := &bytes.Buffer{}
		jpeg.Encode(output, src, &jpeg.Options{
			Quality: 90,
		})

		return output.Bytes(), nil
	}

	var dst draw.Image
	srcX := src.Bounds().Max.X
	srcY := src.Bounds().Max.Y
	srcRatio := float64(srcX) / float64(srcY)
	if maxX == 0 && srcY > maxY {
		newX := int(math.Round(float64(maxY) * srcRatio))
		newY := maxY
		dst = image.NewRGBA(image.Rect(0, 0, newX, newY))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	} else if maxY == 0 && srcX > maxX {
		newX := maxX
		newY := int(math.Round(float64(maxX) / srcRatio))
		dst = image.NewRGBA(image.Rect(0, 0, newX, newY))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	} else if srcX > maxX || srcY > maxY {
		tgtRatio := float64(maxX) / float64(maxY)
		var newX, newY int
		if srcRatio > tgtRatio {
			newX = maxX
			newY = int(math.Round(float64(maxX) / srcRatio))
		} else {
			newX = int(math.Round(float64(maxY) * srcRatio))
			newY = maxY
		}
		dst = image.NewRGBA(image.Rect(0, 0, newX, newY))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, srcX, srcY))
		draw.Copy(dst, image.Point{X: 0, Y: 0}, src, src.Bounds(), draw.Over, nil)
	}

	output := &bytes.Buffer{}
	jpeg.Encode(output, dst, &jpeg.Options{
		Quality: quality,
	})

	return output.Bytes(), nil
}
