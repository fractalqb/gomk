package gomk

import (
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

func Clean(prj *Project, dryrun bool, log *slog.Logger) error {
	prj.Lock()
	defer prj.Unlock()

	for _, g := range prj.Goals(nil) {
		if len(g.ResultOf()) == 0 {
			continue
		}
		if f, ok := g.Artefact.(mkfs.File); ok {
			if ok, err := mkfs.Exists(f, prj); err != nil || !ok {
				continue
			}
			log.Info("remove `artefact`", `artefact`, g.String())
			if !dryrun {
				err := os.Remove(f.Path())
				if err != nil {
					log.Warn(err.Error())
				}
			}
		}
	}
	return nil
}
