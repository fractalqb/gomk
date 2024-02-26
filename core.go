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
