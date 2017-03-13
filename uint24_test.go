package fsbs

import "testing"

func TestTestUint24(t *testing.T) {
	buf := make([]byte, 3)
	for i := uint64(0); i < 1<<24-1; i++ {
		writeInt24(buf, i)
		if readInt24(buf) != i {
			t.Fatal("wrong read at: %d, got %d", i, readInt24(buf))
		}
	}
}
