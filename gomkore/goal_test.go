package gomkore

import (
	"testing"

	"github.com/bits-and-blooms/bitset"
)

func Test_bitset_NextClear_range(t *testing.T) {
	bits := bitset.New(70)
	bits.SetAll().Clear(69)
	cIdx, ok := bits.NextClear(0)
	if !ok {
		t.Error("did not find clear bit")
	}
	if cIdx != 69 {
		t.Errorf("unexpected clear bit %d", cIdx)
	}
	cIdx, ok = bits.NextClear(70)
	if ok {
		t.Errorf("unexpected clear bit %d after end", cIdx)
	}
}
