package superblock

import (
	"bytes"

	"github.com/ipfs/go-sbs/consts"
	uuid "github.com/satori/go.uuid"
)

// Acessort gives raw read access to Superblock (no checks)
type Accessor struct {
	blk []byte
}

// NewAccessor creates new Accessor instance with backing buffer
func NewAccessor(blk []byte) *Accessor {
	return &Accessor{
		blk: blk,
	}
}

func (a *Accessor) checks() error {
	if !bytes.Equal(a.MagicBytes(), []byte(magicBytes)) {
		return errMagicMissMatch
	}
	if a.Version() != 1 {
		return errWrongVersion
	}
	if a.Flags()&reservedMask != 0 {
		return errFlagsReserved
	}

	u := a.UUID()
	if uuid.Equal(u, uuid.Nil) {
		return errUUIDNil
	}

	u2 := a.SecondaryUUID()
	if !uuid.Equal(u, u2) {
		return errUUIDCopyMissMatch
	}

	if a.BlockSize() != consts.BlockSize {
		return errBlockSizeDifferent
	}

	if !isJustZero(a.blk[zero1Start:zero1End]) {
		return errZeroPartIsNotZeroed
	}

	if !isJustZero(a.blk[zero2Start:zero2End]) {
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

func (a *Accessor) MagicBytes() []byte {
	return a.blk[magicStart:magicEnd]
}

// UUID returns UUID os sbs volumene
func (a *Accessor) UUID() uuid.UUID {
	u := uuid.UUID{}
	copy(u[:], a.blk[uuidStart:uuidEnd])
	return u
}

// SecondaryUUID returns backup (recovery) UUID of volumene
func (a *Accessor) SecondaryUUID() uuid.UUID {
	u := uuid.UUID{}
	copy(u[:], a.blk[uuidCopyStart:uuidCopyEnd])
	return u
}

// Version returns the version on Accessor
func (a *Accessor) Version() uint16 {
	return binary.Uint16(a.blk[versionStart:versionEnd])
}

// BlockSize retruns registered blocksize of Accessor
func (a *Accessor) BlockSize() uint32 {
	return binary.Uint32(a.blk[blkSizeStart:blkSizeEnd])
}

// Flags retruns flags field of superblock
func (a *Accessor) Flags() uint16 {
	return binary.Uint16(a.blk[flagsStart:flagsEnd])
}
