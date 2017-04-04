package superblock

import (
	errors "github.com/juju/errors"
)

var (
	errMagicMissMatch      = errors.New("magic bytes different than expected")
	errUUIDCopyMissMatch   = errors.New("copies of UUID and different")
	errZeroPartIsNotZeroed = errors.New("area that should be zero is not")
	errWrongVersion        = errors.New("version is not 1")
	errFlagsReserved       = errors.New("reserved flag is set")
	errBlockSizeDifferent  = errors.New("blockszie different than implemntation")
	errUUIDNil             = errors.New("UUID is Nil")
)
