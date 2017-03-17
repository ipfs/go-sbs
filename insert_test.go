package sbs

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

	dir := sbsDir(t)

	sbs, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < count-100; i++ {
		err := sbs.Put(keys[i], vals[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	err = sbs.Close()
	if err != nil {
		t.Fatal(err)
	}

	sbs, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := count - 100; i < count; i++ {
		err := sbs.Put(keys[i], vals[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	for i, k := range keys {
		val, err := sbs.Get(k)
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
