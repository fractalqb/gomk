package mkfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/testerr"
)

var dirTreeTestFiles = []string{"ls/empty.txt", "ls/sub/empty.xyz"} // Rel to testdata

func TestDirTree_ls(t *testing.T) {
	d := DirTree{Dir: "ls", Filter: IsDir(false)}
	var expect []string
	for _, dttf := range dirTreeTestFiles {
		expect = append(expect, testerr.Shall1(filepath.Rel("ls", dttf)).BeNil(t))
	}
	d.ls("testdata/ls", func(s string, e fs.DirEntry) error {
		if e.IsDir() {
			return nil
		}
		if slices.Index(expect, s) < 0 {
			t.Errorf("unexpected: '%s'", s)
		}
		return nil
	})
}

func TestDirTree_List(t *testing.T) {
	prj := gomkore.NewProject("testdata")
	d := DirTree{Dir: "ls", Filter: IsDir(false)}
	ls := testerr.Shall1(d.List(prj)).BeNil(t)
	if l := len(ls); l != 2 {
		t.Fatalf("ls len: %d", l)
	}
	for _, l := range ls {
		if slices.Index(dirTreeTestFiles, l) < 0 {
			t.Errorf("unexpected ls: %s", l)
		}
	}
}

func TestDirTree_StateAt(t *testing.T) {
	prj := gomkore.NewProject("testdata")
	d := DirTree{Dir: "ls", Filter: IsDir(false)}
	stat := testerr.Shall1(os.Stat("testdata/ls/empty.txt")).BeNil(t)
	et := stat.ModTime()
	stat = testerr.Shall1(os.Stat("testdata/ls/sub/empty.xyz")).BeNil(t)
	if tt := stat.ModTime(); tt.After(et) {
		et = tt
	}
	at := testerr.Shall1(d.StateAt(prj)).BeNil(t)
	if at != et {
		t.Errorf("unexpected mod time %s, want %s", at, et)
	}
}

func TestDirTree_Remove(t *testing.T) {
	prj := gomkore.NewProject("testdata")
	d := DirTree{Dir: "ls", Filter: IsDir(false)}
	cwd := testerr.Shall1(os.Getwd()).BeNil(t)
	emsg := fmt.Sprintf("remove %s/testdata/ls/empty.txt: permission denied", cwd)
	testerr.Shall(d.Remove(prj)).Check(t, testerr.Msg(emsg))
}
