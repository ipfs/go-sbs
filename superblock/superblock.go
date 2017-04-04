package superblock

import (
	"bytes"
	binenc "encoding/binary"

	consts "github.com/ipfs/go-sbs/consts"

	errors "github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	binary binenc.ByteOrder = binenc.LittleEndian
)

// Superblock struct is used as a helper for manipulating the superblock
type Superblock struct {
	blk []byte
}

// Open creates new Superblock structure backed by underlying block.
// The blk has to be BlockSize in size
func Open(blk []byte) (*Superblock, error) {
	if consts.BlockSize != 1<<13 {
		panic("BlockSize has changed, logic in this package might be wrong")
	}

	if len(blk) != consts.BlockSize {
		return nil, errors.Errorf("wrong size passed, expected %d, got %d",
			consts.BlockSize, len(blk))
	}

	s := &Superblock{
		blk: blk,
	}

	if err := s.checks(); err != nil {
		return nil, errors.Trace(err)
	}

	return s, nil
}

func (s *Superblock) checks() error {
	if !bytes.Equal(s.blk[magicStart:magicEnd], []byte(magicBytes)) {
		return errMagicMissMatch
	}
	if s.Version() != 1 {
		return errWrongVersion
	}
	if s.Flags()&reservedMask != 0 {
		return errFlagsReserved
	}

	u := s.UUID()
	if uuid.Equal(u, uuid.Nil) {
		return errUUIDNil
	}

	u2 := s.SecondaryUUID()
	if !uuid.Equal(u, u2) {
		return errUUIDCopyMissMatch
	}

	if s.BlockSize() != consts.BlockSize {
		return errBlockSizeDifferent
	}

	if !isJustZero(s.blk[zero1Start:zero1End]) {
		return errZeroPartIsNotZeroed
	}

	if !isJustZero(s.blk[zero2Start:zero2End]) {
		return errZeroPartIsNotZeroed
	}

	return nil
}

func isJustZero(buf []byte) bool {
	for _, v := range buf {
		if v != byte(0) {
			return false
		}
	}
	return true
}

// UUID returns UUID os sbs volumene
func (s *Superblock) UUID() uuid.UUID {
	u := uuid.UUID{}
	copy(u[:], s.blk[uuidStart:uuidEnd])
	return u
}

// SecondaryUUID returns backup (recovery) UUID of volumene
func (s *Superblock) SecondaryUUID() uuid.UUID {
	u := uuid.UUID{}
	copy(u[:], s.blk[uuidCopyStart:uuidCopyEnd])
	return u
}

// Version returns the version on Superblock
func (s *Superblock) Version() uint16 {
	return binary.Uint16(s.blk[versionStart:versionEnd])
}

// BlockSize retruns registered blocksize of Superblock
func (s *Superblock) BlockSize() uint32 {
	return binary.Uint32(s.blk[blkSizeStart:blkSizeEnd])
}

// Flags retruns flags field of superblock
func (s *Superblock) Flags() uint16 {
	return binary.Uint16(s.blk[flagsStart:flagsEnd])
}
