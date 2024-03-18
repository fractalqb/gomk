package mkfs

import (
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type MkDirs struct {
	MkDirMode fs.FileMode
}

var _ gomkore.Operation = (*MkDirs)(nil)

func (md *MkDirs) Describe(actionHint *gomkore.Action, envHint *gomkore.Env) string {
	return fmt.Sprintf("MkDirs %s", md.MkDirMode)
}

func (md *MkDirs) Do(tr *gomkore.Trace, a *gomkore.Action, env *gomkore.Env) error {
	if md.MkDirMode == 0 {
		tr.Info("MkDirs disabled")
		return nil
	}
	prj := a.Project()
	for _, res := range a.Results() {
		switch res := res.Artefact.(type) {
		case gomkore.Abstract:
			// ignore
		case File:
			path, err := prj.AbsPath(filepath.Dir(res.Path()))
			if err != nil {
				return err
			}
			tr.Debug("create `directory`", `directory`, path)
			return os.MkdirAll(path, md.MkDirMode)
		case Directory:
			path, err := prj.AbsPath(res.Path())
			if err != nil {
				return err
			}
			tr.Debug("create `directory`", `directory`, path)
			return os.MkdirAll(path, md.MkDirMode)
		default:
			return fmt.Errorf("illegal MkDirs result: %T", res)
		}
	}
	return nil
}

func (md *MkDirs) WriteHash(h hash.Hash, a *gomkore.Action, env *gomkore.Env) (bool, error) {
	fmt.Fprintln(h, md.MkDirMode)
	return true, nil
}
