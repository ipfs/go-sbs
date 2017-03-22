package sbs

import (
	"github.com/boltdb/bolt"
	ds "github.com/ipfs/go-datastore"
)

type Sbsds struct {
	sbs  *Sbs
	Path string
}

func NewSbsDS(path string) (*Sbsds, error) {
	sbs, err := Open(path)
	if err != nil {
		return nil, err
	}

	return &Sbsds{
		sbs:  sbs,
		Path: path,
	}, nil
}

func (fs *Sbsds) Put(key ds.Key, value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return ds.ErrInvalidType
	}

	return fs.sbs.Put(key.Bytes(), b)
}

func (fs *Sbsds) Get(key ds.Key) (value interface{}, err error) {
	val, err := fs.sbs.Get(key.Bytes())
	if err == ErrNotFound {
		return nil, ds.ErrNotFound
	}
	return val, err
}

func (fs *Sbsds) Has(key ds.Key) (exists bool, err error) {
	return fs.sbs.Has(key.Bytes())
}

func (fs *Sbsds) Delete(key ds.Key) error {
	err := fs.sbs.Delete(key.Bytes())
	if err == ErrNotFound {
		return ds.ErrNotFound
	}
	return err
}

func (fs *Sbsds) Batch() (ds.Batch, error) {
	return &sbsbatch{
		puts:    make(map[ds.Key][]byte),
		deletes: make(map[ds.Key]struct{}),
		fs:      fs,
	}, nil

}

type sbsbatch struct {
	puts    map[ds.Key][]byte
	deletes map[ds.Key]struct{}

	fs *Sbsds
}

func (bt *sbsbatch) Put(key ds.Key, val interface{}) error {
	b, ok := val.([]byte)
	if !ok {
		return ds.ErrInvalidType
	}

	bt.puts[key] = b
	return nil
}

func (bt *sbsbatch) Delete(key ds.Key) error {
	bt.deletes[key] = struct{}{}
	return nil
}

func (bt *sbsbatch) Commit() error {
	indexData := make(map[ds.Key][]byte)

	for k, val := range bt.puts {
		nblks := blocksNeeded(uint64(len(val)))
		blks, err := bt.fs.sbs.allocateN(nblks)
		if err != nil {
			return err
		}

		bt.fs.sbs.copyToStorage(val, blks)

		data, err := createRecord(val, blks)
		if err != nil {
			return err
		}

		indexData[k] = data
	}

	bt.fs.sbs.index.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketOffset)
		for k, v := range indexData {
			err := b.Put(k.Bytes(), v)
			if err != nil {
				return err
			}
		}
		return nil
	})

	for k, _ := range bt.deletes {
		if err := bt.fs.Delete(k); err != nil {
			return err
		}
	}

	return nil
}

func (fs *Sbsds) Close() error {
	return fs.sbs.Close()
}

var _ ds.Batch = (*sbsbatch)(nil)

var _ ds.Datastore = (*Sbsds)(nil)
var _ ds.Batching = (*Sbsds)(nil)
