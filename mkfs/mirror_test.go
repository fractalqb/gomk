package mkfs

import (
	"testing"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/testerr"
)

func TestMirror_List(t *testing.T) {
	mirr := Mirror{
		Orig:   DirList{Dir: "testdata/ls", Filter: IsDir(false)},
		Strip:  "testdata",
		Dest:   DirTree{Dir: "dest"},
		ExtMap: map[string]string{".txt": ".doc"},
	}
	prj := gomkore.NewProject("")
	ls := testerr.Shall1(mirr.List(prj)).BeNil(t)
	if l := len(ls); l != 1 {
		t.Fatalf("list len %d", l)
	}
	if m := ls[0]; m != "dest/ls/empty.doc" {
		t.Errorf("mirrored: %s", m)
	}
}
