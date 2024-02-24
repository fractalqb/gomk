package gomkore

import (
	"context"
	"hash"
)

// An Action is something you can do in your [Project] to achieve at least one
// [Goal]. The actual implementation of the action is an [Operation]. An action
// without an operation is an "implicit" action, i.e. if all its premises are
// true, all results of the action are implicitly given.
type Action struct {
	Premises []*Goal
	Results  []*Goal
	Op       Operation

	prj *Project
}

func (a *Action) Project() *Project { return a.prj }

func (a *Action) Run(env *Env) error {
	return a.RunContext(context.Background(), env)
}

func (a *Action) RunContext(ctx context.Context, env *Env) error {
	env.Log.Debug("run `action`", `action`, a.String())
	if a.Op == nil {
		return nil
	}
	if env == nil {
		env = DefaultEnv()
	}
	return a.Op.Do(ctx, a, env)
}

func (a *Action) String() string {
	switch {
	case a == nil:
		return "<nil:Action>/" + a.Project().Name(nil)
	case a.Op == nil:
		return "implicit:" + a.Project().Name(nil)
	}
	return a.Op.Describe(a, nil)
}

func (a *Action) WriteHash(h hash.Hash, env *Env) (bool, error) {
	if a.Op == nil {
		return false, nil
	}
	return a.Op.WriteHash(h, a, env)
}

type Operation interface {
	// The hints are optional
	Describe(actionHint *Action, envHint *Env) string
	Do(ctx context.Context, a *Action, env *Env) error
	WriteHash(h hash.Hash, a *Action, env *Env) (bool, error)
}
