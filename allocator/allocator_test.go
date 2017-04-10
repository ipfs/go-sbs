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

func TestResetTip(t *testing.T) {
	_, a := makeAlloc()

	a.tip = 10
	a.ResetTip()
	assert.EqualValues(t, 0, a.tip, "tip after reset is 0")
}

func TestSimpleAlloc(t *testing.T) {
	_, a := makeAlloc()

	start, stop, err := a.Allocate(1)

	assert.EqualValues(t, 1, start,
		"for one block allocations start and stop are equal and be 1")
	assert.EqualValues(t, 1, stop,
		"for one block allocations start and stop are equal and be 1")
	assert.NoError(t, err, "allocating that many blocks should not error")

	start, stop, err = a.Allocate(2)
	assert.EqualValues(t, 2, start, "should have started to allocate second block")
	assert.EqualValues(t, start+1, stop, "end should be start+1")
	assert.NoError(t, err, "there should be no error")

	a.ResetTip()

	start, stop, err = a.Allocate(2)
	assert.EqualValues(t, 4, start, "after reset of tip allocator still works")
	assert.EqualValues(t, start+1, stop, "end should be start+1")
	assert.NoError(t, err, "there should be no error")
}

func TestFullAlloc(t *testing.T) {
	buf, a := makeAlloc()
	_ = buf

	for i := uint(0); i < BlocksPerAllocator-1; i++ {
		start, stop, err := a.Allocate(1)
		assert.EqualValues(t, i+1, start, stop, "start and stop are correct")
		assert.NoError(t, err, "there should be no error in range")
	}

	start, stop, err := a.Allocate(1)
	assert.EqualError(t, err, ErrOutOfSpace.Error(), "full allocator should error")
	assert.Zero(t, start, "in case of full should have zero value")
	assert.Zero(t, stop, "in case of full should have zero value")

	start, stop, err = a.Allocate(1)
	assert.EqualError(t, err, ErrOutOfSpace.Error(), "full allocator should error")
	assert.Zero(t, start, "next alloc in case of full should have zero value")
	assert.Zero(t, stop, "next alloc in case of full should have zero value")

	for _, v := range buf[bitFieldStart:bitFieldEnd] {
		assert.EqualValues(t, 0xff, v, "all space should be filled")
	}
}

func TestAlmostFull(t *testing.T) {
	_, a := makeAlloc()

	_, _, err := a.Allocate(BlocksPerAllocator - 10)
	assert.NoError(t, err, "big allocations should not fail")

	_, _, err = a.Allocate(100)
	assert.EqualError(t, err, ErrOutOfSpace.Error(),
		"big allocation should cause error")

	assert.True(t, a.IsFull(), "allocator should be marked as full")
	assert.True(t, a.Flags()&flagFull != 0, "flag in serialized form is set")
}

func TestFragmented(t *testing.T) {
	_, a := makeAlloc()

	a.setBit(2)

	start, stop, err := a.Allocate(2)
	assert.NoError(t, err, "should not error")
	assert.EqualValues(t, 3, start, "start should be after last set")
	assert.EqualValues(t, 4, stop, "stop should be one bigger than start")
}
