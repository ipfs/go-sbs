package fsbs

import (
	"golang.org/x/crypto/blake2b"
)

const BlkSize = 8192

type Fsbs struct {
	Mem []byte

	shards    map[uint64]*ShardNode
	datas     map[uint64][]byte
	allocPool uint64
}

func Open(path string) (*Fsbs, error) {
	panic("not doing file stuff yet")
}

func Mock() (*Fsbs, error) {
	//return &Fsbs{make([]byte, (8192*BlocksPerAllocator)+10)}, nil
	fsbs := &Fsbs{
		shards:    map[uint64]*ShardNode{},
		allocPool: 2,
		datas:     make(map[uint64][]byte),
	}
	sn := fsbs.NewShardNode()
	fsbs.shards[1] = sn
	return fsbs, nil
}

func (fsbs *Fsbs) Put(k []byte, val []byte) error {
	datablk := fsbs.allocPool
	fsbs.allocPool++
	fsbs.datas[datablk] = val

	h := blake2b.Sum256(k)
	root := fsbs.shards[1]
	return root.insert(h[:], k, &valRef{datablk, uint64(len(val))})
	/*
		if len(val) < valSizeTiny {

		} else if len(val) < BlkSize {

		} else {

		}
	*/
}

func (fsbs *Fsbs) Get(k []byte) ([]byte, error) {
	h := blake2b.Sum256(k)
	root := fsbs.shards[1]
	return root.get(h[:], k)
}

func (fsbs *Fsbs) getShard(i uint64) (*ShardNode, error) {
	return fsbs.shards[i], nil
}

func (fsbs *Fsbs) allocShard() (*ShardNode, uint64, error) {
	v := fsbs.allocPool
	fsbs.allocPool++
	ns := fsbs.NewShardNode()
	fsbs.shards[v] = ns
	return ns, v, nil
}

func (fsbs *Fsbs) printStats() {
	fsbs.shards[1].printStats("0")
}

func (fsbs *Fsbs) density() (uint64, uint64) {
	return fsbs.shards[1].density()
}
