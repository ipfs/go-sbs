package superblock

import (
	"github.com/ipfs/go-sbs/consts"
)

const (
	magicStart    = 1024 // start padding see: https://git.io/vS4Eu
	magicEnd      = magicStart + 16
	uuidStart     = magicEnd
	uuidEnd       = uuidStart + 16
	versionStart  = uuidEnd
	versionEnd    = versionStart + 2
	flagsStart    = versionEnd
	flagsEnd      = flagsStart + 2
	blkSizeStart  = flagsEnd
	blkSizeEnd    = blkSizeStart + 4
	zero1Start    = blkSizeEnd
	zero1End      = consts.BlockSize / 2
	uuidCopyStart = zero1End
	uuidCopyEnd   = uuidCopyStart + 16
	zero2Start    = uuidCopyEnd
	zero2End      = consts.BlockSize
)

const (
	magicBytes = "sbsisablockstore"
)
