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

		if len(val) != len(vals[i]) {
			t.Fatalf("lengths different: %d, %d", len(val), len(vals[i]))
		}
		if !bytes.Equal(val, vals[i]) {
			t.Fatalf("Retrieved data not correct at %d", i)
		}
	}

	os.RemoveAll(dir)
}
