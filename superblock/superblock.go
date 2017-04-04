package superblock

import (
	binenc "encoding/binary"

	consts "github.com/ipfs/go-sbs/consts"

	errors "github.com/juju/errors"
)

var (
	binary binenc.ByteOrder = binenc.LittleEndian
)

// Superblock struct is used as a helper for verifying and acessing the superblock
type Superblock struct {
	*Accessor
}

// Open creates new Superblock structure backed by underlying block.
// The blk has to be BlockSize in size
func OpenSuperblock(blk []byte) (*Superblock, error) {
	if len(blk) != consts.BlockSize {
		return nil, errBlockSizeDifferent
	}

	s := &Superblock{
		Accessor: &Accessor{
			blk: blk,
		},
	}

	if err := s.checks(); err != nil {
		return nil, errors.Trace(err)
	}

	return s, nil
}
