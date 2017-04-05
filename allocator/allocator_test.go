package allocator

import (
	"testing"

	"github.com/ipfs/go-sbs/consts"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
)

func makeBlk() []byte {
	return make([]byte, consts.BlockSize)
}

func makeAlloc() ([]byte, *Allocator) {
	blk := makeBlk()
	err := FormatAllocator(blk, uuid.NewV4())
	if err != nil {
		panic(err.Error())
	}

	return blk, OpenAllocator(blk)
}

func TestUUID(t *testing.T) {
	blk := makeBlk()
	err := FormatAllocator(blk, uuid.Nil)
	assert.NoError(t, err, "Format should not error")
}

func TestIncTip(t *testing.T) {
	_, a := makeAlloc()

	assert.EqualValues(t, 0, a.tip, "tip should be 0 after creation")
	err := a.incTip()
	assert.NoError(t, err, "inc from 0 should not fail")
	assert.EqualValues(t, 1, a.tip, "this should be 1 after increase")

	a.tip = BlocksPerAllocator - 2
	err = a.incTip()
	assert.NoError(t, err, "inc from almost max should not fail")
	assert.EqualValues(t, BlocksPerAllocator-1, a.tip,
		"this should be at limit after increase")

	err = a.incTip()
	assert.EqualError(t, err, ErrOutOfSpace.Error(),
		"should raise out of space error")
	assert.EqualValues(t, BlocksPerAllocator-1, a.tip,
		"this should be at limit after increase")
}

func TestIncTipByByte(t *testing.T) {
	_, a := makeAlloc()

	assert.EqualValues(t, 0, a.tip, "tip should be 0 after creation")
	err := a.incTipByByte()
	assert.NoError(t, err, "inc from 0 should not fail")
	assert.EqualValues(t, 8, a.tip, "this should be 1 after increase")

	a.tip = BlocksPerAllocator - 9
	err = a.incTipByByte()
	assert.NoError(t, err, "inc from almost max should not fail")
	assert.EqualValues(t, BlocksPerAllocator-8, a.tip,
		"this should be at limit after increase")

	err = a.incTipByByte()
	assert.EqualError(t, err, ErrOutOfSpace.Error(),
		"should raise out of space error")
	assert.EqualValues(t, BlocksPerAllocator-8, a.tip,
		"this should be at limit after increase")
}
