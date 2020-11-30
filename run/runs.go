package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Infof("%s %s", name, strings.Join(arg, " "))
	if err := cmd.Run(); err != nil {
		log.Panice(err)
	}
}

type Dirs struct {
	pre *Dirs
	dir string
}

func WdDirs() (Dirs, error) {
	d, err := os.Getwd()
	if err != nil {
		return Dirs{}, err
	}
	return Dirs{dir: d}, nil
}

func MustWdDirs() Dirs {
	res, err := WdDirs()
	if err != nil {
		log.Panice(err)
	}
	return res
}

func (d Dirs) Name(elem ...string) string {
	res := filepath.Join(append([]string{d.dir}, elem...)...)
	res = filepath.Clean(res)
	return res
}

func (d Dirs) Push(elem ...string) Dirs {
	tmp := d.Name(elem...)
	return Dirs{pre: &d, dir: tmp}
}

func (d Dirs) Pop(levels int) *Dirs {
	res := &d
	for levels > 0 {
		res = res.pre
		if res == nil {
			break
		}
		levels--
	}
	return res
}
