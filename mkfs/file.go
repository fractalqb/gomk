package mkfs

import (
	"os"
	"path/filepath"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type File string

var _ Artefact = File("")

func (f File) Path() string { return string(f) }

func (f File) Name(in *gomkore.Project) string {
	n, _ := in.RelPath(f.Path())
	return n
}

func (f File) StateAt(in *gomkore.Project) time.Time {
	rp, err := in.RelPath(f.Path())
	if err != nil {
		return time.Time{}
	}
	st, err := os.Stat(rp)
	if err != nil || st.IsDir() {
		return time.Time{}
	}
	return st.ModTime()
}

// func (f File) Rel(in *gomkore.Project) (File {
// 	path := in.RelPath(f.Path())
// 	return File(filepath.ToSlash(path))
// }

func (f File) Moved(strip, dest Directory) (File, error) {
	var path string
	if strip == nil {
		var err error
		path, err = fsMove(f.Path(), "", dest.Path())
		if err != nil {
			return File(""), err
		}
	} else {
		var err error
		path, err = fsMove(f.Path(), strip.Path(), dest.Path())
		if err != nil {
			return File(""), err
		}
	}
	return File(filepath.ToSlash(path)), nil
}

func (f File) Ext() string { return filepath.Ext(f.Path()) }

func (f File) WithExt(ext string) File {
	path := f.Path()
	if ext == "" {
		ext = filepath.Ext(path)
		if ext == "" {
			return f
		}
		return File(path[:len(path)-len(ext)])
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	fExt := filepath.Ext(path)
	if fExt == "" {
		return File(path + ext)
	}
	return File(path[:len(path)-len(fExt)] + ext)
}
