package gomk

import (
	"log/slog"
	"os"
)

func Clean(prj *Project, dryrun bool, log *slog.Logger) error {
	prj.Lock()
	defer prj.Unlock()

	for _, g := range prj.Goals(nil) {
		if len(g.ResultOf) == 0 {
			continue
		}
		switch a := g.Artefact.(type) {
		case File:
			if !a.Exists(prj) {
				continue
			}
			log.Info("remove `artefact`", `artefact`, g.String())
			if !dryrun {
				err := os.Remove(a.Path())
				if err != nil {
					log.Warn(err.Error())
				}
			}
		}
		// TODO clean Directory???
	}
	return nil
}
