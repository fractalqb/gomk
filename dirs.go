package gomk

import (
	"path/filepath"
)

type WDir struct {
	b   *Build
	pre *WDir
	dir string
}

func (d *WDir) Build() *Build { return d.b }

func (d *WDir) Join(elem ...string) string {
	return filepath.Join(append([]string{d.dir}, elem...)...)
}

func (d *WDir) Rel(elem ...string) (string, error) {
	tmp := d.Join(elem...)
	return d.Build().Rel(tmp)
}

func (d *WDir) MustRel(path string) string {
	res, err := d.Rel(path)
	if err != nil {
		panic(err)
	}
	return res
}

func (d *WDir) Cd(dirs ...string) *WDir {
	tmp := make([]string, len(dirs)+1)
	tmp[0] = d.dir
	copy(tmp[1:], dirs)
	return &WDir{
		b:   d.b,
		pre: d,
		dir: filepath.Join(tmp...),
	}
}

func (d *WDir) Back() *WDir { return d.pre }
