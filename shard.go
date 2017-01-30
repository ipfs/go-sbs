package fsbs

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/blake2b"
)

var ErrNotFound = fmt.Errorf("not found")

const (
	RFEmpty = iota
	RFTiny
	RFDirect
	RFRange
	RFTrie

	Chained = 1 << 6
	IsChain = 1 << 5
)

const ChainingLength = 10

type ShardNode struct {
	fsbs *Fsbs

	Size       int
	Depth      int
	KeyRecords []KeyRecord

	tempkeys [][]byte
}

func (fsbs *Fsbs) NewShardNode() *ShardNode {
	return &ShardNode{
		fsbs:       fsbs,
		KeyRecords: make([]KeyRecord, 256),
		tempkeys:   make([][]byte, 256),
	}
}

type KeyRecord struct {
	Flag   byte
	Size   uint64
	Block  uint64
	Offset uint16
	Shard  uint64
}

func (sn *ShardNode) storeKey(i int, key []byte) error {
	sn.tempkeys[i] = key
	return nil
}

func (sn *ShardNode) getKey(i int) ([]byte, error) {
	return sn.tempkeys[i], nil
}

type valRef struct {
	Block uint64
	Size  uint64
}

func (sn *ShardNode) get(h, key []byte) ([]byte, error) {
	r := &sn.KeyRecords[h[0]]
	if r.Flag == 0 {
		if r.Shard != 0 {
			panic("should not have empty record with shard attached")
		}
		return nil, ErrNotFound
	}

	k, err := sn.getKey(int(h[0]))
	if err != nil {
		return nil, err
	}
	if bytes.Equal(k, key) {
		return sn.fsbs.datas[r.Block], nil
	}

	if r.Flag&Chained != 0 {
		if r.Shard != 0 {
			panic("should not have shard on a chained entry")
		}
		for i := 2; i < ChainingLength; i++ {
			ix := (int(h[0]) + (i * i)) % 256
			r := &sn.KeyRecords[ix]
			if r.Flag == 0 {
				return nil, ErrNotFound
			}

			k, err := sn.getKey(ix)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(k, key) {
				return sn.fsbs.datas[r.Block], nil
			}
		}
		return nil, ErrNotFound
	}

	child, err := sn.fsbs.getShard(r.Shard)
	if err != nil {
		return nil, err
	}

	return child.get(h[1:], key)
}

func (sn *ShardNode) reInsert(key []byte, val *valRef) error {
	h := blake2b.Sum256(key)
	return sn.insert(h[sn.Depth:], key, val)
}

func (sn *ShardNode) setVal(ix int, key []byte, val *valRef, flag byte) error {
	r := &sn.KeyRecords[ix]
	r.Flag = (RFDirect | flag) // only doing direct values for now, whatever
	r.Block = val.Block
	r.Size = val.Size
	if err := sn.storeKey(ix, key); err != nil {
		return err
	}
	sn.Size++
	return nil
}

func (sn *ShardNode) insert(h, key []byte, val *valRef) error {
	r := &sn.KeyRecords[h[0]]
	if r.Flag&IsChain != 0 {
		okey, err := sn.getKey(int(h[0]))
		if err != nil {
			return err
		}

		oval := &valRef{Block: r.Block, Size: r.Size}
		sn.Size-- // drop size since we're pulling it out to reinsert

		if err := sn.setVal(int(h[0]), key, val, 0); err != nil {
			return err
		}

		return sn.reInsert(okey, oval)
	}

	if r.Flag == 0 {
		return sn.setVal(int(h[0]), key, val, 0)
	}

	if r.Shard == 0 {
		for i := 2; i < ChainingLength; i++ {
			ix := (int(h[0]) + (i * i)) % 256
			cr := &sn.KeyRecords[ix]
			if cr.Flag == 0 {
				r.Flag |= Chained
				return sn.setVal(ix, key, val, IsChain)
			}
		}
	}

	var ss *ShardNode
	if r.Shard != 0 {
		s, err := sn.fsbs.getShard(r.Shard)
		if err != nil {
			return err
		}
		ss = s
	} else {
		s, i, err := sn.fsbs.allocShard()
		if err != nil {
			return err
		}
		s.Depth = sn.Depth + 1

		r.Shard = i
		ss = s
	}

	if r.Flag&Chained != 0 {
		r.Flag &^= Chained
		// Undo chaining
		for i := 2; i < ChainingLength; i++ {
			ix := (int(h[0]) + (i * i)) % 256
			cr := &sn.KeyRecords[ix]
			if cr.Flag&IsChain == 0 {
				continue
			}

			k, err := sn.getKey(ix)
			if err != nil {
				return err
			}

			val := &valRef{
				Block: cr.Block,
				Size:  cr.Size,
			}

			sn.Size--
			cr.Flag = 0
			if err := sn.reInsert(k, val); err != nil {
				return err
			}
		}
	}

	return ss.insert(h[1:], key, val)
}

func (sn *ShardNode) printStats(s string) {
	fmt.Printf("node %s: size = %d\n", s, sn.Size)
	for i, r := range sn.KeyRecords {
		if r.Flag == 0 {
			continue
		}
		if r.Shard != 0 {
			child, err := sn.fsbs.getShard(r.Shard)
			if err != nil {
				panic(err)
			}

			child.printStats(fmt.Sprintf("%s-%d", s, i))
		}
		k, err := sn.getKey(i)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%s-%d: %s\n", s, i, string(k))
	}
}

func (sn *ShardNode) density() (uint64, uint64) {
	have := uint64(sn.Size)
	total := uint64(256)
	for _, r := range sn.KeyRecords {
		if r.Shard != 0 {
			child, err := sn.fsbs.getShard(r.Shard)
			if err != nil {
				panic(err)
			}

			ch, ct := child.density()
			have += ch
			total += ct
		}
	}

	return have, total
}
