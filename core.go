package gomk

import (
	"errors"
	"fmt"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type (
	Env     = gomkore.Env
	Project = gomkore.Project
	Goal    = gomkore.Goal
	Action  = gomkore.Action

	Abstract = gomkore.Abstract
)

func DefaultEnv() *Env { return gomkore.DefaultEnv() }

func NewProject(dir string) *Project { return gomkore.NewProject(dir) }

// Edit calls do with wrappers of [gomkore] types that allow easy editing of
// project definitions. Edit recovers from any panic and returns it as an error,
// so the idiomatic error handling within do can be skipped.
func Edit(prj *Project, do func(ProjectEd)) (err error) {
	prj.Lock()
	defer func() {
		prj.Unlock()
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			case string:
				err = errors.New(p)
			default:
				err = fmt.Errorf("panic: %+v", p)
			}
		}
	}()
	do(ProjectEd{prj})
	return
}

const (
	UpdAllActions  gomkore.UpdateMode = 0
	UpdSomeActions gomkore.UpdateMode = 1
	UpdAnyAction   gomkore.UpdateMode = 2
	UpdOneAction   gomkore.UpdateMode = 3

	UpdUnordered gomkore.UpdateMode = 4
)
