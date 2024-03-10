package gomk

import (
	"os"

	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

func FsGoals(prj ProjectEd, dir, dirTmpl mkfs.Directory) (goals []GoalEd) {
	ls := mustRet(dir.List(prj.Project()))
	for _, e := range ls {
		pe := prj.RelPath(e)
		st := mustRet(os.Stat(pe))
		if st.IsDir() {
			switch dirTmpl := dirTmpl.(type) {
			case mkfs.DirList:
				dir := dirTmpl
				dir.Dir = e
				g := prj.Goal(dir)
				goals = append(goals, g)
			case mkfs.DirTree:
				dir := dirTmpl
				dir.Dir = e
				g := prj.Goal(dir)
				goals = append(goals, g)
			}
		} else {
			g := prj.Goal(mkfs.File(e))
			goals = append(goals, g)
		}
	}
	return goals
}
