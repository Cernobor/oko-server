package server

import (
	"cernobor.cz/oko-server/models"
)

func ptrInt(x int) *int {
	return &x
}

func ptrInt64(x int64) *int64 {
	return &x
}

func ptrString(x string) *string {
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
