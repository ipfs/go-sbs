package superblock

import (
	"testing"

	"github.com/ipfs/go-sbs/consts"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestManualSuperblock(t *testing.T) {
	buf, _, _ := tSupBlk()
	errMsgStart := "should have caught "

	s, err := OpenSuperblock(buf)

	assert.EqualError(t, err, errMagicMissMatch.Error(),
		errMsgStart+"lack of magic bytes")
	assert.Nil(t, s, "should be nil")

	// put magic
	copy(buf[magicStart:magicEnd], []byte(magicBytes))
	s, err = OpenSuperblock(buf)

	assert.EqualError(t, err, errWrongVersion.Error(),
		errMsgStart+"wrong version")
	assert.Nil(t, s, "should be nil")

	binary.PutUint16(buf[versionStart:versionEnd], 1)
	s, err = OpenSuperblock(buf)

	assert.EqualError(t, err, errUUIDNil.Error(),
		errMsgStart+"nil UUID")
	assert.Nil(t, s, "should be nil")

	u := uuid.NewV4()
	copy(buf[uuidStart:uuidEnd], u[:])
	s, err = OpenSuperblock(buf)

	assert.EqualError(t, err, errUUIDCopyMissMatch.Error(),
		errMsgStart+"UUID missmatch")
	assert.Nil(t, s, "should be nil")

	copy(buf[uuidCopyStart:uuidCopyEnd], u[:])
	s, err = OpenSuperblock(buf)

	assert.EqualError(t, err, errBlockSizeDifferent.Error(),
		errMsgStart+"blocksize missmatch")
	assert.Nil(t, s, "should be nil")

	binary.PutUint32(buf[blkSizeStart:blkSizeEnd], consts.BlockSize)
	s, err = OpenSuperblock(buf)
	assert.NotNil(t, s, "Superblock should been loaded")
	assert.NoError(t, err, "no error should been given")
}

func TestZeroBlockIsChecked(t *testing.T) {
	blk, _, _ := tSupBlk()
	Format(blk)

	s, err := OpenSuperblock(blk)
	assert.NoError(t, err, "opening should succeed")
	assert.NotNil(t, s, "Superblock should exist")

	for i := zero1Start; i < zero1End; i++ {
		blk[i] = byte(5)
		s, err = OpenSuperblock(blk)
		assert.EqualError(t, err, errZeroPartIsNotZeroed.Error(), "should be detected")
		assert.Nil(t, s, "Superblock should not be created")

		blk[i] = byte(0)
	}

	for i := zero2Start; i < zero2End; i++ {
		blk[i] = byte(5)
		s, err = OpenSuperblock(blk)
		assert.EqualError(t, err, errZeroPartIsNotZeroed.Error(), "should be detected")
		assert.Nil(t, s, "Superblock should not be created")

		blk[i] = byte(0)
	}
}

func TestWrongLenghtIsDetected(t *testing.T) {
	for i := 1000; i < consts.BlockSize*10; i += 300 {
		blk := make([]byte, i)

		s, err := OpenSuperblock(blk)
		assert.EqualError(t, err, errBlockSizeDifferent.Error(),
			"open with wrong size should fiail")
		assert.Nil(t, s, "Superblock should not be created")
	}
}

func TestReservedFlags(t *testing.T) {
	blk, _, w := tSupBlk()
	err := Format(blk)
	assert.NoError(t, err, "Format should work")
	for i := 1; i != 1<<15; i = i << 1 {
		badFlag := i & reservedMask
		if badFlag == 0 {
			continue
		}

		w.SetFlags(uint16(badFlag))

		s, err := OpenSuperblock(blk)
		assert.EqualError(t, err, errFlagsReserved.Error(),
			"open with wrong flag should fiail")
		assert.Nil(t, s, "Superblock should not be created")

		w.SetFlags(0)

	}
}
