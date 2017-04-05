package allocator

import (
	"github.com/ipfs/go-sbs/consts"
)

const (
	uuidStart     = 0
	uuidEnd       = uuidStart + 16
	flagsStart    = uuidEnd
	flagsEnd      = flagsStart + 2
	reservedStart = flagsEnd
	reservedEnd   = reservedStart + 14
	bitFieldStart = reservedEnd
	bitFieldEnd   = consts.BlockSize
)

const (
	headerStart = uuidStart
	headerEnd   = reservedEnd

	AllocatorHeaderSize = headerEnd - headerStart
	BlocksPerAllocator  = (consts.BlockSize - AllocatorHeaderSize) * 8
)

const (
	flagFull = 1 << iota
	flagFragmented
	// insert flags here

	lastFlag
)

const (
	reservedMask = (1<<16 - 1) & ^(lastFlag - 1)
)
