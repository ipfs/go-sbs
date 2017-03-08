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

type Fsbs struct {
	Mem []byte

	mmfi  *os.File
	mm    mmap.MMap
	index *bolt.DB

	alloc    *AllocatorBlock
	curAlloc *AllocatorBlock

	curTx  *bolt.Tx
	bucket *bolt.Bucket
}

func Open(path string) (*Fsbs, error) {
	datapath := filepath.Join(path, "data")
	indexpath := filepath.Join(path, "index")

	db, err := bolt.Open(indexpath, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("data"))
		return err
	})
	if err != nil {
		return nil, err
	}

	fi, err := os.Open(datapath)
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
	if err := fsbs.curTx.Commit(); err != nil {
		return err
	}

	if err := fsbs.index.Close(); err != nil {
		return err
	}

	if err := fsbs.mm.Unmap(); err != nil {
		return err
	}

	return nil
}

func (fsbs *Fsbs) expand() error {
	err := fsbs.mmfi.Truncate(int64(fsbs.curAlloc.Offset + (2 * (BlockSize * BlocksPerAllocator))))
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

	offs := fsbs.curAlloc.Offset + BlocksPerAllocator
	nalloc, err := LoadAllocator(fsbs.mm[offs*BlockSize : (offs+1)*BlockSize])
	if err != nil {
		return err
	}

	nalloc.Offset = offs
	fsbs.curAlloc = nalloc
	return nil
}

func (fsbs *Fsbs) Put(k []byte, val []byte) error {
	nblks := uint64(len(val)) / BlockSize
	if uint64(len(val))%BlockSize != 0 {
		nblks++
	}

	blks, err := fsbs.curAlloc.Allocate(nblks)
	switch err {
	case ErrAllocatorFull:
		if err := fsbs.expand(); err != nil {
			return err
		}

		mblks, err := fsbs.curAlloc.Allocate(nblks - uint64(len(blks)))
		if err != nil {
			return err
		}
		blks = append(blks, mblks...)
	default:
		return err
	case nil:
	}

	for i, blk := range blks {
		beg := i * BlockSize
		end := (i + 1) * BlockSize
		if len(val)-beg < end {
			end = len(val)
		}
		copy(fsbs.mm[blk*BlockSize:(blk+1)*BlockSize], val[beg:end])
	}

	t := pb.Record_Indirect
	rec := &pb.Record{
		Blocks: blks,
		Size_:  proto.Uint64(uint64(len(val))),
		Type:   &t,
	}

	data, err := proto.Marshal(rec)
	if err != nil {
		return err
	}

	buck, err := fsbs.getBucket()
	if err != nil {
		return err
	}
	return buck.Put(k, data)
}

func (fsbs *Fsbs) getBucket() (*bolt.Bucket, error) {
	if fsbs.bucket == nil {
		tx, err := fsbs.index.Begin(true)
		if err != nil {
			return nil, err
		}
		fsbs.curTx = tx
		fsbs.bucket = tx.Bucket([]byte("data"))
	}

	return fsbs.bucket, nil
}

func (fsbs *Fsbs) Get(k []byte) ([]byte, error) {
	rec := fsbs.bucket.Get(k)
	if rec == nil {
		return nil, ErrNotFound
	}

	var prec pb.Record
	if err := proto.Unmarshal(rec, &prec); err != nil {
		return nil, err
	}

	out := make([]byte, prec.GetSize_())
	var beg uint64
	for _, blk := range prec.GetBlocks() {
		l := uint64(BlockSize)
		if uint64(len(out))-beg < l {
			l = uint64(len(out)) - beg
		}
		blkoff := blk * BlockSize
		copy(out[beg:beg+l], fsbs.mm[blkoff:blkoff+l])
		beg += l
	}
	return out, nil
}

func (fsbs *Fsbs) Delete(k []byte) error {
	rec := fsbs.bucket.Get(k)
	if rec == nil {
		return ErrNotFound
	}

	var prec pb.Record
	if err := proto.Unmarshal(rec, &prec); err != nil {
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
	curs := fsbs.bucket.Cursor()
	return curs.Next
}
