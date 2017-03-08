package fsbs

import (
	"encoding/binary"
	"fmt"
)

var ErrAllocatorFull = fmt.Errorf("allocator full")

const (
	FlagFragmented = 1 << iota
)

const (
	AllocatorHeaderSize = 64
	BlocksPerAllocator  = (BlockSize - AllocatorHeaderSize) * 8
)

type AllocatorBlock struct {
	Version       int
	InUse         uint64
	LastAllocator uint64
	Offset        uint64
	Flag          byte
	FreeBlocks    int
	FreeBlockList []int
	Bitfield      []byte
	buf           []byte
}

func readInt24(buf []byte) uint64 {
	return uint64(buf[2]) + uint64(buf[1]<<8) + uint64(buf[0]<<16)
}

func writeInt24(buf []byte, v uint64) {
	buf[2] = byte(v) & 0xff
	buf[1] = byte(v>>8) & 0xff
	buf[0] = byte(v>>16) & 0xff
}

func InitAllocator(buf []byte) {
	buf[0] = 1
	buf[1] = 0
	buf[2] = 0
	buf[3] = 1
}

func LoadAllocator(buf []byte) (*AllocatorBlock, error) {
	a := new(AllocatorBlock)
	a.Version = int(buf[0])
	a.Bitfield = buf[64:]
	if a.Version != 1 {
		InitAllocator(buf)
		a.SetBit(0)
	}
	a.InUse = readInt24(buf[1:4])
	a.LastAllocator = binary.BigEndian.Uint64(buf[4:12])
	a.buf = buf
	return a, nil
}

func (a *AllocatorBlock) SetBit(i uint64) error {
	ix := i / 8
	pos := uint(i % 8)
	a.Bitfield[ix] = a.Bitfield[ix] | (1 << pos)
	return nil
}

func (a *AllocatorBlock) ClearBit(i uint64) error {
	ix := i / 8
	pos := uint(i % 8)
	a.Bitfield[ix] &^= (1 << pos)
	return nil
}

func (a *AllocatorBlock) Allocate(n uint64) ([]uint64, error) {
	if a.Flag&FlagFragmented != 0 {
		panic("cant handle fragmented allocation yet")
	}

	if a.InUse == BlocksPerAllocator {
		return nil, ErrAllocatorFull
	}

	if n+a.InUse > BlocksPerAllocator {
		panic("cant yet handle allocations past allocator bounds")
	}

	var out []uint64
	for i := a.InUse; i < a.InUse+n; i++ {
		err := a.SetBit(i)
		if err != nil {
			return nil, err
		}
		out = append(out, i+a.Offset)
	}
	a.InUse += n
	writeInt24(a.buf[1:4], a.InUse)

	return out, nil
}

func (a *AllocatorBlock) Free(blks []uint64) error {
	for _, b := range blks {
		a.ClearBit(b)
	}
	return nil
}
