package fsbs

import (
	"github.com/boltdb/bolt"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

type Fsbsds struct {
	fsbs *Fsbs
	Path string
}

func NewFsbsDS(path string) (ds.Batching, error) {
	fsbs, err := Open(path)
	if err != nil {
		return nil, err
	}

	return &Fsbsds{
		fsbs: fsbs,
		Path: path,
	}, nil
}

func (fs *Fsbsds) Put(key ds.Key, value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return ds.ErrInvalidType
	}

	return fs.fsbs.Put(key.Bytes(), b)
}

func (fs *Fsbsds) Get(key ds.Key) (value interface{}, err error) {
	return fs.fsbs.Get(key.Bytes())
}

func (fs *Fsbsds) Has(key ds.Key) (exists bool, err error) {
	return fs.fsbs.Has(key.Bytes())
}

func (fs *Fsbsds) Delete(key ds.Key) error {
	return fs.fsbs.Delete(key.Bytes())
}

func (fs *Fsbsds) Query(q query.Query) (query.Results, error) {
	panic("not implemented")
}

func (fs *Fsbsds) Batch() (ds.Batch, error) {
	return &fsbsbatch{
		puts:    make(map[ds.Key][]byte),
		deletes: make(map[ds.Key]struct{}),
		fs:      fs,
	}, nil

}

type fsbsbatch struct {
	puts    map[ds.Key][]byte
	deletes map[ds.Key]struct{}

	fs *Fsbsds
}

func (bt *fsbsbatch) Put(key ds.Key, val interface{}) error {
	b, ok := val.([]byte)
	if !ok {
		return ds.ErrInvalidType
	}

	bt.puts[key] = b
	return nil
}

func (bt *fsbsbatch) Delete(key ds.Key) error {
	bt.deletes[key] = struct{}{}
	return nil
}

func (bt *fsbsbatch) Commit() error {
	indexData := make(map[ds.Key][]byte)

	for k, val := range bt.puts {
		nblks := blocksNeeded(uint64(len(val)))
		blks, err := bt.fs.fsbs.allocateN(nblks)
		if err != nil {
			return err
		}

		bt.fs.fsbs.copyToStorage(val, blks)

		data, err := createRecord(val, blks)
		if err != nil {
			return err
		}

		indexData[k] = data
	}

	bt.fs.fsbs.index.Update(func(tx *bolt.Tx) error {
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

var _ ds.Batch = (*fsbsbatch)(nil)

var _ ds.Batching = (*Fsbsds)(nil)
