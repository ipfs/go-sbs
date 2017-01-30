package fsbs

import (
	"encoding/binary"
)

const (
	FlagFragmented = 1 << iota
)

const (
	AllocatorHeaderSize = 64
	BlocksPerAllocator  = (BlockSize - AllocatorHeaderSize) * 8
)

type AllocatorBlock struct {
	Version       int
	InUse         int
	LastAllocator uint64
	Flag          byte
	FreeBlocks    int
	FreeBlockList []int
	Bitfield      []byte
}

func readInt(buf []byte) int {
	out := int(buf[len(buf)-1])
	for i := 1; i < len(buf); i++ {
		out += int(buf[len(buf)-(1+i)]) << (8 * uint(i))
	}
	return out
}

func LoadAllocator(buf []byte) (*AllocatorBlock, error) {
	a := new(AllocatorBlock)
	a.Version = int(buf[0])
	a.InUse = readInt(buf[1:3])
	a.LastAllocator = binary.BigEndian.Uint64(buf[4:12])
	a.Bitfield = buf[64:]
	return a, nil
}

func (a *AllocatorBlock) SetBit(i int) error {
	ix := i / 8
	pos := uint(i % 8)
	a.Bitfield[ix] = a.Bitfield[ix] | (1 << pos)
	return nil
}

func (a *AllocatorBlock) Allocate(n int) ([]int, error) {
	if a.Flag&FlagFragmented != 0 {
		panic("cant handle fragmented allocation yet")
	}

	if n+a.InUse > BlocksPerAllocator {
		panic("cant yet handle allocations past allocator bounds")
	}

	var out []int
	for i := a.InUse; i < a.InUse+n; i++ {
		err := a.SetBit(i)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	a.InUse += n

	return out, nil
}

func (a *AllocatorBlock) Free(blks []int) error {
	panic("cant free blocks yet, what the hell do we even do?")
}
