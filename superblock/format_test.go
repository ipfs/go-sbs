package superblock

import (
	"testing"

	"github.com/ipfs/go-sbs/consts"
	"github.com/stretchr/testify/assert"
)

func TestFormatIsValidSuperblock(t *testing.T) {
	blk := make([]byte, consts.BlockSize)

	err := Format(blk)

	assert.NoError(t, err, "format should not error")
}
