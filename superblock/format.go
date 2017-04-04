package superblock

import (
	"github.com/ipfs/go-sbs/consts"
	uuid "github.com/satori/go.uuid"
)

func Format(blk []byte) error {
	w := &Writer{
		blk: blk,
	}
	u := uuid.NewV4()

	w.SetUUID(u)
	w.SetVersion(1)
	w.SetFlags(0)
	w.SetBlocksize(consts.BlockSize)
	w.ZeroOutZeros()

	return nil
}
