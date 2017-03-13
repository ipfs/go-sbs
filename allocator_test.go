package fsbs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"
)

var seed int64 = -1
var seedSlab []byte
var seedBlocks int = 10
var randBlockMax int = 8

func init() {
	seedSlab = make([]byte, BlockSize*(10+1))
	if seed == -1 {
		seed = time.Now().UTC().UnixNano()
	}
	rnd := rand.New(rand.NewSource(seed))

	fmt.Printf("test seed is: %d\n", seed)

	_, err := rnd.Read(seedSlab)
	if err != nil {
		panic(err)
	}
}

func lerp(a, b int, x float64) int {
	return a + int(float64(b)*x)
}

type rng struct {
	index  int
	kindex int
}

func (rng *rng) inc() {
	rng.index = (rng.index + 1) % (seedBlocks - randBlockMax)
}

func (rng *rng) getRandBlock() []byte {
	x := float64(seedSlab[rng.index]) / 255
	rng.inc()
	size := lerp(BlockSize/2, BlockSize*randBlockMax, x)

	defer rng.inc()
	return seedSlab[rng.index : rng.index+size]
}

func (rng *rng) getRandKey() []byte {
	defer func() {
		rng.kindex = rng.kindex + 1
	}()
	return seedSlab[rng.index : rng.index+32]
}

func fsbsDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "fsbs")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAllocatorOverrideTest(t *testing.T) {
	rng := rng{}

	dir := fsbsDir(t)
	t.Log("fsbs dir:", dir)
	fsbs, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	vals := make(map[string][]byte)

	for i := 0; i < 10; i++ {
		k := rng.getRandKey()
		v := rng.getRandBlock()
		vals[string(k)] = v
		err = fsbs.Put(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = fsbs.Close()
	if err != nil {
		t.Fatal(err)
	}

	fsbs, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// check
	for k, v := range vals {
		key := []byte(k)
		value, err := fsbs.Get(key)
		if err != nil {
			t.Fatal("got error", err)
		}
		if !bytes.Equal(v, value) {
			t.Fatal("data not equal")
		}
	}

	// Put some more
	for i := 0; i < 10; i++ {
		k := rng.getRandKey()
		v := rng.getRandBlock()
		err = fsbs.Put(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	// recheck
	for k, v := range vals {
		key := []byte(k)
		value, err := fsbs.Get(key)
		if err != nil {
			t.Fatal("got error", err)
		}
		if !bytes.Equal(v, value) {
			t.Fatal("data not equal")
		}
	}

	err = fsbs.Close()
	if err != nil {
		t.Fatal(err)
	}

}
