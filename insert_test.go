package fsbs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestInserting(t *testing.T) {
	count := 70000 // large enough that we fill the first allocator
	var keys [][]byte
	var vals [][]byte
	for i := 0; i < count; i++ {
		keys = append(keys, []byte(fmt.Sprintf("key%d", i)))
		vals = append(vals, []byte(fmt.Sprintf("val%d", i)))
	}

	dir, err := ioutil.TempDir("", "fsbs")
	if err != nil {
		t.Fatal(err)
	}

	fsbs, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < count; i++ {
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
}
