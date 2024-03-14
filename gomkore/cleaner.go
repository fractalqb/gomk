package gomkore

import (
	"time"
)

func Clean(prj *Project, dryrun bool, tr *Trace) error {
	prj.LockBuild()
	defer prj.Unlock()
	start := time.Now()
	tr = tr.pushProject(prj)
	tr.startProject(prj, "cleaning")
	for _, g := range prj.Goals(nil) {
		if len(g.ResultOf()) == 0 {
			continue
		}
		if f, ok := g.Artefact.(RemovableArtefact); ok && g.Removable {
			str := tr.pushGoal(g)
			if ok, err := f.Exists(prj); err != nil || !ok {
				continue
			}
			str.removeArtefact(g)
			if !dryrun {
				err := f.Remove(prj)
				if err != nil {
					tr.Warn(err.Error())
				}
			}
		}
	}
	tr.doneProject(prj, "cleaning", time.Since(start))
	return nil
}

type CleanTracer interface {
	TracerCommon

	RemoveArtefact(*Trace, *Goal)
}
