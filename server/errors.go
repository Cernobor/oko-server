package server

type ConstError string

func (e ConstError) Error() string {
	return string(e)
}

const (
	ErrUserNotExists       = ConstError("user does not exist")
	ErrUserAlreadyExists   = ConstError("user already exists")
	ErrAttemptedSystemUser = ConstError("attepted to associate with system user")
)
