package pack

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
	root  string
	stack []string
}

func WdDirs() (Dirs, error) {
	d, err := os.Getwd()
	if err != nil {
		return Dirs{}, err
	}
	return Dirs{root: d}, nil
}

func MustWdDirs() Dirs {
	res, err := WdDirs()
	if err != nil {
		log.Panice(err)
	}
	return res
}

func (d *Dirs) In(dir string, do func()) {
	d.Pushd(dir)
	defer d.Popd()
	do()
}

func (d *Dirs) InEach(do func(), dirs ...string) {
	for _, dir := range dirs {
		d.In(dir, do)
	}
}

func (d *Dirs) Pushd(dir string) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatale(err)
	}
	d.stack = append(d.stack, wd)
	if err = os.Chdir(dir); err != nil {
		log.Fatale(err)
	}
	log.Infoa("enter `directory`", dir)
}

func (d *Dirs) Popd() string {
	if len(d.stack) == 0 {
		log.Fatals("empty dir stack")
	}
	sl := len(d.stack)
	bd := d.stack[sl-1]
	d.stack = d.stack[:sl-1]
	if err := os.Chdir(bd); err != nil {
		log.Fatale(err)
	}
	if rel, err := filepath.Rel(d.root, bd); err != nil {
		log.Infoa("return to `directory`", bd)
	} else {
		log.Infoa("return to `directory`", rel)
	}
	return bd

}
