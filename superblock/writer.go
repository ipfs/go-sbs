package superblock

import (
	uuid "github.com/satori/go.uuid"
)

// Writer allows to modify the Superblock, it shouldn't be use light
type Writer struct {
	blk []byte
}

// NewWriter creates new Writer instance
func NewWriter(blk []byte) *Writer {
	return &Writer{
		blk: blk,
	}
}

// SetUUID writes both Primary and Secondary UUID to Superblock
func (w *Writer) SetUUID(u uuid.UUID) {
	copy(w.blk[uuidStart:uuidEnd], u[:])
	copy(w.blk[uuidCopyStart:uuidCopyEnd], u[:])
}

func (w *Writer) SetVersion(v uint16) {
	binary.PutUint16(w.blk[versionStart:versionEnd], v)
}

func (w *Writer) SetFlags(f uint16) {
	binary.PutUint16(w.blk[flagsStart:flagsEnd], f)
}

func (w *Writer) SetBlocksize(bsize uint32) {
	binary.PutUint32(w.blk[blkSizeStart:blkSizeEnd], bsize)
}

func (w *Writer) ZeroOutZeros() {
	for i, _ := range w.blk[zero1Start:zero1End] {
		w.blk[i] = byte(0)
	}

	for i, _ := range w.blk[zero2Start:zero2End] {
		w.blk[i] = byte(0)
	}
}
