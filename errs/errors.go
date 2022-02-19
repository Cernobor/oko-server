package errs

import "fmt"

type ConstError string

func (e ConstError) Error() string {
	return string(e)
}

const (
	ErrUserNotExists               = ConstError("user does not exist")
	ErrUserAlreadyExists           = ConstError("user already exists")
	ErrPhotoNotExists              = ConstError("photo does not exist (for the given feature)")
	ErrAttemptedSystemUser         = ConstError("attempted to associate with system user")
	ErrUnsupportedImageFormat      = ConstError("unsupported image format")
	ErrUnsupportedImageContentType = ConstError("unsupported image content type")
	ErrNonUniqueCreatedFeatureIDs  = ConstError("created features do not have unique IDs")
)

type EErrPhotoNotProvided *ErrPhotoNotProvided

type ErrPhotoNotProvided struct {
	Reference string
}

func NewErrPhotoNotProvided(reference string) *ErrPhotoNotProvided {
	return &ErrPhotoNotProvided{
		Reference: reference,
	}
}

func (e *ErrPhotoNotProvided) Error() string {
	return fmt.Sprintf("referenced photo %s which was not provided", e.Reference)
}

type EErrPhotoThumbnailNotProvided *ErrPhotoThumbnailNotProvided

type ErrPhotoThumbnailNotProvided struct {
	Reference string
}

func NewErrPhotoThumbnailNotProvided(reference string) *ErrPhotoThumbnailNotProvided {
	return &ErrPhotoThumbnailNotProvided{
		Reference: reference,
	}
}

func (e *ErrPhotoThumbnailNotProvided) Error() string {
	return fmt.Sprintf("referenced photo %s the thumbnail of which was not provided", e.Reference)
}

type ErrFeatureForPhotoNotExists struct {
	PhotoFeatureReference int64
}

func NewErrFeatureForPhotoNotExists(reference int64) *ErrFeatureForPhotoNotExists {
	return &ErrFeatureForPhotoNotExists{
		PhotoFeatureReference: reference,
	}
}

func (e *ErrFeatureForPhotoNotExists) Error() string {
	return fmt.Sprintf("photos were referenced for feature with local ID %d which does not exist in posted created features", e.PhotoFeatureReference)
}

type ErrUnsupportedContentType struct {
	Reference string
}

func NewErrUnsupportedContentType(reference string) *ErrUnsupportedContentType {
	return &ErrUnsupportedContentType{
		Reference: reference,
	}
}

func (e *ErrUnsupportedContentType) Error() string {
	return fmt.Sprintf("photo %s has an unsupported Content-Type", e.Reference)
}
