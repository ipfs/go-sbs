package fsbs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	mmap "github.com/edsrzf/mmap-go"
	proto "github.com/gogo/protobuf/proto"
	pb "github.com/ipfs/fsbs/pb"
)

var ErrNotFound = fmt.Errorf("not found")

const BlockSize = 8192

var (
	bucketOffset = []byte("offsets")
)

type Fsbs struct {
	Mem []byte

	mmfi  *os.File
	mm    mmap.MMap
	index *bolt.DB

	alloc    *AllocatorBlock
	curAlloc *AllocatorBlock
}

func Open(path string) (*Fsbs, error) {
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
		err = fi.Truncate(int64(BlockSize * BlocksPerAllocator))
		if err != nil {
			return nil, err
		}
	}

	mm, err := mmap.Map(fi, mmap.RDWR, 0)
	if err != nil {
		return nil, err
	}

	alloc, err := LoadAllocator(mm[:BlockSize])
	if err != nil {
		return nil, err
	}

	return &Fsbs{
		mmfi:     fi,
		mm:       mm,
		index:    db,
		alloc:    alloc,
		curAlloc: alloc,
	}, nil
}

func (fsbs *Fsbs) Close() error {
	if err := fsbs.index.Close(); err != nil {
		return err
	}

	if err := fsbs.mm.Unmap(); err != nil {
		return err
	}

	return nil
}

func (fsbs *Fsbs) nextAllocator() error {
	currEnd := fsbs.curAlloc.Offset + BlocksPerAllocator
	newEnd := currEnd + BlocksPerAllocator
	if uint64(len(fsbs.mm)) < newEnd*BlockSize {
		err := fsbs.expand()
		if err != nil {
			return err
		}
	}

	nalloc, err := LoadAllocator(fsbs.mm[currEnd*BlockSize : newEnd*BlockSize])
	if err != nil {
		return err
	}
	nalloc.Offset = currEnd
	fsbs.curAlloc = nalloc

	_ = fsbs.mm[newEnd*BlockSize-1] // range check

	return nil

}

func (fsbs *Fsbs) expand() error {
	currEnd := fsbs.curAlloc.Offset + BlocksPerAllocator
	newEnd := int64(currEnd + BlocksPerAllocator)

	err := fsbs.mmfi.Truncate(newEnd * BlockSize)
	if err != nil {
		return err
	}

	err = fsbs.mm.Unmap()
	if err != nil {
		return err
	}

	nmm, err := mmap.Map(fsbs.mmfi, mmap.RDWR, 0)
	if err != nil {
		return err
	}

	fsbs.mm = nmm

	return nil
}

func blocksNeeded(length uint64) uint64 {
	nblks := length / BlockSize
	if length%BlockSize != 0 {
		nblks++
	}
	return nblks
}

func (fsbs *Fsbs) allocateN(nblks uint64) ([]uint64, error) {
	blks, err := fsbs.curAlloc.Allocate(nblks)

	defer func() {
		fmt.Printf("blks: %v\n", blks)
	}()

	switch err {
	case ErrAllocatorFull:
		if err := fsbs.nextAllocator(); err != nil {
			return nil, err
		}

		mblks, err := fsbs.curAlloc.Allocate(nblks - uint64(len(blks)))
		if err != nil {
			return nil, err
		}
		blks = append(blks, mblks...)
		fallthrough
	case nil:
		return blks, nil
	default:
		return nil, err
	}
}

func (fsbs *Fsbs) copyToStorage(val []byte, blks []uint64) {
	for i, blk := range blks {
		l := BlockSize
		beg := i * BlockSize

		if bufleft := len(val) - beg; bufleft < l {
			l = bufleft
		}
		//fmt.Printf("trying to write: %d, blocklen: %d", blk, len(fsbs.mm)/BlockSize)
		blkoff := blk * BlockSize
		copy(fsbs.mm[blkoff:blkoff+uint64(l)], val[beg:beg+l])
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

func (fsbs *Fsbs) Put(k []byte, val []byte) error {
	nblks := blocksNeeded(uint64(len(val)))
	blks, err := fsbs.allocateN(nblks)
	if err != nil {
		return err
	}
	data, err := createRecord(val, blks)
	if err != nil {
		return err
	}

	fsbs.copyToStorage(val, blks)

	err = fsbs.index.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketOffset)
		return b.Put(k, data)
	})

	return err
}

func (fsbs *Fsbs) getPB(k []byte) (*pb.Record, error) {
	var prec pb.Record

	err := fsbs.index.View(func(tx *bolt.Tx) error {
		rec := tx.Bucket(bucketOffset).Get(k)
		if len(rec) == 0 {
			return ErrNotFound
		}

		err := proto.Unmarshal(rec, &prec)
		return err
	})
	return &prec, err
}

func (fsbs *Fsbs) Has(k []byte) (bool, error) {
	has := false
	err := fsbs.index.View(func(tx *bolt.Tx) error {
		rec := tx.Bucket(bucketOffset).Get(k)
		if len(rec) == 0 {
			has = true
		}
		return nil
	})
	return has, err
}

func (fsbs *Fsbs) Get(k []byte) ([]byte, error) {
	prec, err := fsbs.getPB(k)
	if err != nil {
		return nil, err
	}

	out := make([]byte, prec.GetSize_())
	var beg uint64
	for _, blk := range prec.GetBlocks() {
		l := uint64(BlockSize)
		if lsize := uint64(len(out)) - beg; lsize < l {
			l = lsize
		}
		blkoff := blk * BlockSize
		copy(out[beg:beg+l], fsbs.mm[blkoff:blkoff+l])
		beg += l
	}
	return out, nil
}

func (fsbs *Fsbs) Delete(k []byte) error {
	var prec pb.Record

	err := fsbs.index.Update(func(tx *bolt.Tx) error {
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
		wa := blk / BlocksPerAllocator
		wi := blk % BlocksPerAllocator
		tofree[wa] = append(tofree[wa], wi)
	}

	for wa, list := range tofree {
		beg := wa * BlockSize * BlocksPerAllocator
		alloc, err := LoadAllocator(fsbs.mm[beg : beg+BlockSize])
		if err != nil {
			return err
		}

		if err := alloc.Free(list); err != nil {
			return err
		}
	}
	return nil
}

func (fsbs *Fsbs) GetIterator() func() ([]byte, []byte) {
	return nil
}
