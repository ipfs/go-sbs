package fsbs

import (
	"bytes"
	"fmt"
	"testing"
)

func TestInserting(t *testing.T) {
	count := 100000
	var keys [][]byte
	var vals [][]byte
	for i := 0; i < count; i++ {
		keys = append(keys, []byte(fmt.Sprintf("key%d", i)))
		vals = append(vals, []byte(fmt.Sprintf("val%d", i)))
	}

	fsbs, _ := Mock()
	for i := 0; i < count; i++ {
		err := fsbs.Put(keys[i], vals[i])
		if err != nil {
			t.Fatal(err)
		}

		out, err := fsbs.Get(keys[i])
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(vals[i], out) {
			t.Fatal("mismatch", i)
		}

		if i%100 == 0 {
			a, b := fsbs.density()
			fmt.Println(i, a, b)
		}
	}

	for i, k := range keys {
		val, err := fsbs.Get(k)
		if err != nil {
			t.Fatalf("key %d: %s", i, err)
		}
		if !bytes.Equal(val, vals[i]) {
			t.Fatal("Retrieved data not correct")
		}
	}

}
