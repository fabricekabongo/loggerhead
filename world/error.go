package world

import (
	"errors"
)

var (
	TreeErrLocationNil         = errors.New("insertion failed because location is nil")
	TreeErrLocationOutOfBounds = errors.New("insertion failed because location is out of bounds")
)
