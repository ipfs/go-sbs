package superblock

import (
	"math"
	"testing"

	"github.com/ipfs/go-sbs/consts"
	"github.com/stretchr/testify/assert"
)

func tSupBlk() ([]byte, *Accessor, *Writer) {
	blk := make([]byte, consts.BlockSize)
	a := NewAccessor(blk)
	w := NewWriter(blk)
	return blk, a, w
}

func TestWriterMagicBytes(t *testing.T) {
	_, a, w := tSupBlk()

	w.SetMagicBytes()
	magic := a.MagicBytes()
	assert.Equal(t, magic, []byte(magicBytes))
}

func TestWriterVersion(t *testing.T) {
	_, a, w := tSupBlk()
	for i := 0; i <= math.MaxUint16; i++ {
		s := uint16(i)
		w.SetVersion(s)
		v := a.Version()
		assert.Equal(t, s, v, "version read is not the same as written")
	}
}

func TestWriterBlockSize(t *testing.T) {
	_, a, w := tSupBlk()
	for i := 1; i <= math.MaxUint32/2; i = i * 2 {
		s := uint32(i)
		w.SetBlocksize(s)
		v := a.BlockSize()
		assert.Equal(t, s, v, "version read is not the same as written")
	}
}
