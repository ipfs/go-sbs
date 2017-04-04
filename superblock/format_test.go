package superblock

import (
	"testing"

	"github.com/ipfs/go-sbs/consts"
	"github.com/stretchr/testify/assert"
)

func TestFormatIsValidSuperblock(t *testing.T) {
	blk := make([]byte, consts.BlockSize)

	for i, _ := range blk {
		blk[i] = byte(i & 0xff)
	}

	err := Format(blk)
	assert.NoError(t, err, "format should not error")

	a := NewAccessor(blk)
	assert.Equal(t, a.Flags(), uint16(0), "flags should not be set")

	s, err := OpenSuperblock(blk)
	assert.NoError(t, err, "format should have returned valid superblock")
	if !assert.NotNil(t, s, "superblock should be valid") {
		a := NewAccessor(blk)
		t.Logf("magic: %s (%x)", a.MagicBytes(), a.MagicBytes())
	}

}
