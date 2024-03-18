package gomk

import (
	"context"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

func NewBuilder(tr *gomkore.Trace, env *gomkore.Env) *gomkore.Builder {
	if tr == nil {
		tr = gomkore.NewTrace(context.Background(), NewDefaultTracer())
	}
	res, _ := gomkore.NewBuilder(tr, env)
	return res
}

func NewChanger(tr *gomkore.Trace, env *gomkore.Env) *gomkore.Changer {
	if tr == nil {
		tr = gomkore.NewTrace(context.Background(), NewDefaultTracer())
	}
	res, _ := gomkore.NewChanger(tr, env)
	return res
}

func Clean(prj *gomkore.Project, dryrun bool, tr *gomkore.Trace) error {
	if tr == nil {
		tr = gomkore.NewTrace(context.Background(), NewDefaultTracer())
	}
	return gomkore.Clean(prj, dryrun, tr)
}
