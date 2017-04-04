package superblock

import (
	"testing"

	"github.com/ipfs/go-sbs/consts"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestManualSuperblock(t *testing.T) {
	buf := make([]byte, consts.BlockSize)
	errMsgStart := "should have caught "

	s, err := Open(buf)

	assert.EqualError(t, err, errMagicMissMatch.Error(),
		errMsgStart+"lack of magic bytes")
	assert.Nil(t, s, "should be nil")

	// put magic
	copy(buf[magicStart:magicEnd], []byte(magicBytes))
	s, err = Open(buf)

	assert.EqualError(t, err, errWrongVersion.Error(),
		errMsgStart+"wrong version")
	assert.Nil(t, s, "should be nil")

	binary.PutUint16(buf[versionStart:versionEnd], 1)
	s, err = Open(buf)

	assert.EqualError(t, err, errUUIDNil.Error(),
		errMsgStart+"nil UUID")
	assert.Nil(t, s, "should be nil")

	u := uuid.NewV4()
	copy(buf[uuidStart:uuidEnd], u[:])
	s, err = Open(buf)

	assert.EqualError(t, err, errUUIDCopyMissMatch.Error(),
		errMsgStart+"UUID missmatch")
	assert.Nil(t, s, "should be nil")

	copy(buf[uuidCopyStart:uuidCopyEnd], u[:])
	s, err = Open(buf)

	assert.EqualError(t, err, errBlockSizeDifferent.Error(),
		errMsgStart+"blocksize missmatch")
	assert.Nil(t, s, "should be nil")

	binary.PutUint32(buf[blkSizeStart:blkSizeEnd], consts.BlockSize)
	s, err = Open(buf)
	assert.NotNil(t, s, "Superblock should been loaded")
	assert.NoError(t, err, "no error should been given")
}
