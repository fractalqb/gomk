package mkfs

import (
	"fmt"
	"io/fs"
	"os"
	"testing"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/testerr"
)

func TestDirList_ls(t *testing.T) {
	d := DirList{Dir: "ls", Filter: IsDir(false)}
	d.ls("testdata/ls", func(p string, e fs.DirEntry) error {
		if e.IsDir() {
			return nil
		}
		if p != "empty.txt" {
			t.Errorf("unexpected: '%s'", p)
		}
		return nil
	})
}

func TestDirList_List(t *testing.T) {
	prj := gomkore.NewProject("testdata")
	d := DirList{Dir: "ls", Filter: IsDir(false)}
	ls := testerr.F1(d.List(prj)).ShallBeNil(t)
	if l := len(ls); l != 1 {
		t.Fatalf("ls len: %d", l)
	}
	if e := ls[0]; e != "ls/empty.txt" { // Rel to project
		t.Fatalf("ls: %s", e)
	}
}

func TestDirList_StateAt(t *testing.T) {
	prj := gomkore.NewProject("testdata")
	d := DirList{Dir: "ls", Filter: IsDir(false)}
	stat := testerr.F1(os.Stat("testdata/ls/empty.txt")).ShallBeNil(t)
	at := testerr.F1(d.StateAt(prj)).ShallBeNil(t)
	if at != stat.ModTime() {
		t.Errorf("unexpected mod time %s, want %s", at, stat.ModTime())
	}
}

func TestDirList_Remove(t *testing.T) {
	prj := gomkore.NewProject("testdata")
	d := DirList{Dir: "ls", Filter: IsDir(false)}
	cwd := testerr.F1(os.Getwd()).ShallBeNil(t)
	emsg := fmt.Sprintf("remove %s/testdata/ls/empty.txt: permission denied", cwd)
	testerr.F0(d.Remove(prj)).ShallMsg(t, emsg)
}
