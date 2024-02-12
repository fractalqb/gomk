package gomk

import (
	"log/slog"
	"os"
)

func Clean(prj *Project, dryrun bool, log *slog.Logger) error {
	prj.buildLock.Lock()
	defer prj.buildLock.Unlock()

	for _, g := range prj.goals {
		if len(g.ResultOf) == 0 {
			continue
		}
		switch a := g.Artefact.(type) {
		case File:
			if !a.Exists() {
				continue
			}
			log.Info("remove `artefact`", `artefact`, g.String())
			if !dryrun {
				err := os.Remove(a.Path())
				if err != nil {
					log.Warn(err.Error())
				}
			}
		case Directory:

		}
	}
	return nil
}
