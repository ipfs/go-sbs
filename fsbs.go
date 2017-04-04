package sbs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-sbs/consts"
	pb "github.com/ipfs/go-sbs/pb"

	"github.com/boltdb/bolt"
	proto "github.com/gogo/protobuf/proto"
	mmap "github.com/gxed/mmap-go"
)

var ErrNotFound = fmt.Errorf("not found")

var (
	bucketOffset = []byte("offsets")
)

type Sbs struct {
	Mem []byte

	mmfi  *os.File
	mm    mmap.MMap
	index *bolt.DB

	alloc    *AllocatorBlock
	curAlloc *AllocatorBlock
}

func Open(path string) (*Sbs, error) {
	datapath := filepath.Join(path, "data")
	indexpath := filepath.Join(path, "index")

	db, err := bolt.Open(indexpath, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketOffset)
		return err
	})
	if err != nil {
		return nil, err
	}

	fi, err := os.OpenFile(datapath, os.O_RDWR, 0300)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		fi, err = os.Create(datapath)
		if err != nil {
			return nil, err
		}
		err = fi.Truncate(int64(consts.BlockSize * consts.BlocksPerAllocator))
		if err != nil {
			return nil, err
		}
	}

	mm, err := mmap.Map(fi, mmap.RDWR, 0)
	if err != nil {
		return nil, err
	}

	alloc, err := LoadAllocator(mm[:consts.BlockSize])
	if err != nil {
		return nil, err
	}

	return &Sbs{
		mmfi:     fi,
		mm:       mm,
		index:    db,
		alloc:    alloc,
		curAlloc: alloc,
	}, nil
}

func (sbs *Sbs) Close() error {
	if err := sbs.index.Close(); err != nil {
		return err
	}

	if err := sbs.mm.Unmap(); err != nil {
		return err
	}

	return nil
}

func (sbs *Sbs) nextAllocator() error {
	currEnd := sbs.curAlloc.Offset + consts.BlocksPerAllocator
	newEnd := currEnd + consts.BlocksPerAllocator
	if uint64(len(sbs.mm)) < newEnd*consts.BlockSize {
		err := sbs.expand()
		if err != nil {
			return err
		}
	}

	nalloc, err := LoadAllocator(sbs.mm[currEnd*consts.BlockSize : newEnd*consts.BlockSize])
	if err != nil {
		return err
	}
	nalloc.Offset = currEnd
	sbs.curAlloc = nalloc

	_ = sbs.mm[newEnd*consts.BlockSize-1] // range check

	return nil

}

func (sbs *Sbs) expand() error {
	currEnd := sbs.curAlloc.Offset + consts.BlocksPerAllocator
	newEnd := int64(currEnd + consts.BlocksPerAllocator)

	err := sbs.mmfi.Truncate(newEnd * consts.BlockSize)
	if err != nil {
		return err
	}

	err = sbs.mm.Unmap()
	if err != nil {
		return err
	}

	nmm, err := mmap.Map(sbs.mmfi, mmap.RDWR, 0)
	if err != nil {
		return err
	}

	sbs.mm = nmm

	return nil
}

func blocksNeeded(length uint64) uint64 {
	nblks := length / consts.BlockSize
	if length%consts.BlockSize != 0 {
		nblks++
	}
	return nblks
}

func (sbs *Sbs) allocateN(nblks uint64) ([]uint64, error) {
	blks := make([]uint64, 0, nblks)

	for uint64(len(blks)) != nblks {
		mblks, err := sbs.curAlloc.Allocate(nblks - uint64(len(blks)))
		switch err {
		case ErrAllocatorFull:
			err = sbs.nextAllocator()
			if err != nil {
				return nil, err
			}
			fallthrough
		case nil:
			blks = append(blks, mblks...)
		default:
			return nil, err
		}
	}

	return blks, nil
}

func (sbs *Sbs) copyToStorage(val []byte, blks []uint64) {
	for i, blk := range blks {
		l := consts.BlockSize
		beg := i * consts.BlockSize

		if bufleft := len(val) - beg; bufleft < l {
			l = bufleft
		}
		//fmt.Printf("trying to write: %d, blocklen: %d", blk, len(sbs.mm)/consts.BlockSize)
		blkoff := blk * consts.BlockSize
		copy(sbs.mm[blkoff:blkoff+uint64(l)], val[beg:beg+l])
	}
}

func createRecord(val []byte, blks []uint64) ([]byte, error) {
	t := pb.Record_Indirect
	rec := &pb.Record{
		Blocks: blks,
		Size_:  proto.Uint64(uint64(len(val))),
		Type:   &t,
	}

	return proto.Marshal(rec)
}

func (sbs *Sbs) Put(k []byte, val []byte) error {
	nblks := blocksNeeded(uint64(len(val)))
	blks, err := sbs.allocateN(nblks)
	if err != nil {
		return err
	}
	data, err := createRecord(val, blks)
	if err != nil {
		return err
	}

	sbs.copyToStorage(val, blks)

	err = sbs.index.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketOffset)
		return b.Put(k, data)
	})

	return err
}

func (sbs *Sbs) getPB(k []byte) (*pb.Record, error) {
	var prec pb.Record

	err := sbs.index.View(func(tx *bolt.Tx) error {
		rec := tx.Bucket(bucketOffset).Get(k)
		if len(rec) == 0 {
			return ErrNotFound
		}

		err := proto.Unmarshal(rec, &prec)
		return err
	})
	return &prec, err
}

func (sbs *Sbs) Has(k []byte) (bool, error) {
	has := false
	err := sbs.index.View(func(tx *bolt.Tx) error {
		rec := tx.Bucket(bucketOffset).Get(k)
		if len(rec) != 0 {
			has = true
		}
		return nil
	})
	return has, err
}

func (sbs *Sbs) read(prec *pb.Record, out []byte) {
	var beg uint64
	for _, blk := range prec.GetBlocks() {
		l := uint64(consts.BlockSize)
		if lsize := uint64(len(out)) - beg; lsize < l {
			l = lsize
		}
		blkoff := blk * consts.BlockSize
		copy(out[beg:beg+l], sbs.mm[blkoff:blkoff+l])
		beg += l
	}
}

func (sbs *Sbs) Get(k []byte) ([]byte, error) {
	prec, err := sbs.getPB(k)
	if err != nil {
		return nil, err
	}

	out := make([]byte, prec.GetSize_())
	sbs.read(prec, out)
	return out, nil
}

func (sbs *Sbs) Delete(k []byte) error {
	var prec pb.Record

	err := sbs.index.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketOffset)
		rec := b.Get(k)
		if len(rec) == 0 {
			return ErrNotFound
		}
		err := b.Delete(k)
		if err != nil {
			return err
		}

		return proto.Unmarshal(rec, &prec)
	})
	if err != nil {
		return err
	}

	tofree := make(map[uint64][]uint64)
	for _, blk := range prec.GetBlocks() {
		wa := blk / consts.BlocksPerAllocator
		wi := blk % consts.BlocksPerAllocator
		tofree[wa] = append(tofree[wa], wi)
	}

	for wa, list := range tofree {
		beg := wa * consts.BlockSize * consts.BlocksPerAllocator
		alloc, err := LoadAllocator(sbs.mm[beg : beg+consts.BlockSize])
		if err != nil {
			return err
		}

		if err := alloc.Free(list); err != nil {
			return err
		}
	}
	return nil
}
