package mkfs

import (
	"os"
	"path/filepath"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type DirPath struct {
	Dir    string
	NoStat bool
}

var _ Artefact = DirPath{}

func (d DirPath) Name(in *gomkore.Project) string {
	n, _ := in.RelPath(d.Dir)
	return n
}

func (d DirPath) StateAt(in *gomkore.Project) (time.Time, error) {
	if d.NoStat {
		return time.Time{}, nil
	}
	prjDir, err := in.AbsPath(d.Path())
	if err != nil {
		return time.Time{}, err
	}
	st, err := os.Stat(prjDir)
	if err != nil {
		return time.Time{}, err
	}
	return st.ModTime(), nil
}

func (d DirPath) Path() string { return d.Dir }

func (d DirPath) Moved(strip, dest Directory) (File, error) {
	var path string
	if strip == nil {
		var err error
		path, err = fsMove(d.Path(), "", dest.Path())
		if err != nil {
			return File(""), err
		}
	} else {
		var err error
		path, err = fsMove(d.Path(), strip.Path(), dest.Path())
		if err != nil {
			return File(""), err
		}
	}
	return File(filepath.ToSlash(path)), nil
}
