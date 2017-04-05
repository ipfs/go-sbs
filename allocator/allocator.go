package allocator

import (
	binenc "encoding/binary"

	"github.com/juju/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	binary binenc.ByteOrder = binenc.LittleEndian
)

type Allocator struct {
	blk      []byte
	bitfield []byte

	tip uint
}

func OpenAllocator(blk []byte) *Allocator {
	return &Allocator{
		blk:      blk,
		bitfield: blk[bitFieldStart:bitFieldEnd],
		tip:      0,
	}
}

func FormatAllocator(blk []byte, u uuid.UUID) error {
	for i, _ := range blk {
		blk[i] = byte(0)
	}
	copy(blk[uuidStart:uuidEnd], u[:])
	aloc := OpenAllocator(blk)

	// paranoia check
	readUUID := aloc.UUID()
	if !uuid.Equal(u, readUUID) {
		panic("uuid not equal, this SHOULD NOT happen")
	}

	aloc.setBit(0)

	return nil
}

func (a *Allocator) UUID() uuid.UUID {
	u := uuid.UUID{}
	copy(u[:], a.blk[uuidStart:uuidEnd])
	return u
}

func (a *Allocator) Flags() uint16 {
	return binary.Uint16(a.blk[flagsStart:flagsEnd])
}

func (a *Allocator) SetFlags(flags uint16) {
	binary.PutUint16(a.blk[flagsStart:flagsEnd], flags)
}

func (a *Allocator) IsFull() bool {
	return a.Flags()&flagFull != 0
}

func (a *Allocator) setFull() {
	a.SetFlags(a.Flags() | flagFull)
}

func (a *Allocator) ResetTip() {
	a.tip = 0
}

func (a *Allocator) incTip() error {
	new := a.tip + 1
	if new == BlocksPerAllocator {
		return ErrOutOfSpace
	}
	a.tip = new
	return nil
}

func (a *Allocator) incTipByByte() error {
	new := (a.tip + 8) & ^uint(7) // move to next byte
	if new == BlocksPerAllocator {
		return ErrOutOfSpace
	}
	a.tip = new
	return nil
}

func (a *Allocator) setBit(i uint) {
	ix := i / 8
	pos := uint(i % 8)
	a.bitfield[ix] = a.bitfield[ix] | (1 << pos)
}

func (a *Allocator) getBit(i uint) bool {
	ix := i / 8
	pos := uint(i % 8)
	return a.bitfield[ix]&(1<<pos) != 0
}

func (a *Allocator) clearBit(i uint) {
	ix := i / 8
	pos := uint(i % 8)
	a.bitfield[ix] &^= (1 << pos)
}

func (a *Allocator) Allocate(count uint) (uint, uint, error) {
	if a.IsFull() {
		return 0, 0, errors.Trace(ErrOutOfSpace)
	}

outer:
	for {
		var start, end uint
		// Skip by a byte at a time
		for a.bitfield[a.tip/8] == 0xff {
			if err := a.incTipByByte(); err != nil {
				a.setFull()
				return 0, 0, errors.Trace(err)
			}
		}

		for !a.getBit(a.tip) {
			if err := a.incTip(); err != nil {
				a.setFull()
				return 0, 0, errors.Trace(err)
			}
		}
		start = a.tip

		for end-start != count {
			if err := a.incTip(); err != nil {
				a.setFull()
				return 0, 0, errors.Trace(err)
			}
			if !a.getBit(a.tip) {
				continue outer
			}
			end = a.tip
		}
		return start, end, nil
	}

}
