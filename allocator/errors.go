package allocator

import (
	errors "github.com/juju/errors"
)

var (
	ErrOutOfSpace = errors.New("allocator is out of space")
)
