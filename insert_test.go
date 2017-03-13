package fsbs

import (
	"bytes"
	"os"
	"testing"
)

func TestInserting(t *testing.T) {
	rng := rng{}
	count := BlockSize * 11

	keys := make([][]byte, 0, count)
	vals := make([][]byte, 0, count)

	for i := 0; i < count; i++ {
		keys = append(keys, rng.getRandKey())
		vals = append(vals, rng.getRandBlock())
	}

	dir := fsbsDir(t)

	fsbs, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < count-10; i++ {
		err := fsbs.Put(keys[i], vals[i])
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

	for i := count - 10; i < count; i++ {
		err := fsbs.Put(keys[i], vals[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	for i, k := range keys {
		val, err := fsbs.Get(k)
		if err != nil {
			t.Fatalf("key %d: %s", i, err)
		}
		if !bytes.Equal(val, vals[i]) {
			t.Fatal("Retrieved data not correct", i, val, vals[i])
		}
	}

	os.RemoveAll(dir)
}
