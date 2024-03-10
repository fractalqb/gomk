package gomk

import (
	"context"
	"errors"
	"fmt"
	"hash"

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

// Goals is meant to be used when implementing [Operation] to select and check
// linked goals gs.
//
// See also [Tangible], [AType]
func Goals(gs []*Goal, exclusive bool, matchAll ...func(*Goal) bool) ([]*Goal, error) {
	mLen1 := len(matchAll) - 1
	res := make([]*Goal, 0, len(gs))
NEXT_GOAL:
	for gi, g := range gs {
		for pi, pred := range matchAll {
			if !pred(g) {
				if exclusive && pi == mLen1 {
					return nil, fmt.Errorf("illegal goal %d: %s", gi, g.Name())
				}
				continue NEXT_GOAL
			}
		}
		res = append(res, g)
	}
	return res, nil
}

func Tangible(g *Goal) bool { return !g.IsAbstract() }

func AType[A gomkore.Artefact](g *Goal) bool {
	_, ok := g.Artefact.(A)
	return ok
}

func OpFunc(desc string, f func(context.Context, *Action, *Env) error) gomkore.Operation {
	return funcOp{desc: desc, f: f}
}

type funcOp struct {
	desc string
	f    func(context.Context, *Action, *Env) error
}

func (fo funcOp) Describe(*Action, *Env) string {
	return fo.desc
}

func (fo funcOp) Do(ctx context.Context, a *Action, env *Env) error {
	env.Log.Debug("call `function`", `function`, fo.desc)
	return fo.f(ctx, a, env)
}

func (fo funcOp) WriteHash(hash.Hash, *Action, *Env) (bool, error) {
	return false, nil
}

func mustEd(err error) {
	if err != nil {
		panic(err)
	}
}

func mustRet[T any](v T, err error) T {
	mustEd(err)
	return v
}
