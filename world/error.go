package world

import (
	"errors"
)

var (
	ErrTreeLocationNil         = errors.New("insertion failed because location is nil")
	ErrTreeLocationOutOfBounds = errors.New("insertion failed because location is out of bounds")
)
