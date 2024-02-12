package gomk

import (
	"context"
	"fmt"
)

// An Action is something you can do in your [Project] to achive a [Goal]. The
// actual implementation of the action is an [Operation]. An action without an
// operation is an "implicit" action, i.e. if all its premises are true, all
// results of the action are implicitly given.
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

// Valid returns an error if a's [Operation] is a bad-operation. The error
// describes what is wrong with action a.
func (a *Action) Valid() error {
	if bop, ok := a.Op.(badOp); ok {
		if bop.op == nil {
			return fmt.Errorf("bad operation:%w", bop.err)
		}
		return fmt.Errorf("bad operation:%s:%w",
			bop.op.Describe(a.Project()),
			bop.err,
		)
	}
	return nil
}

func (a *Action) String() string {
	switch {
	case a == nil:
		return "<nil:Action>/" + a.Project().Name(nil)
	case a.Op == nil:
		return "implicit:" + a.Project().Name(nil)
	}
	return a.Op.Describe(a.Project())
}

type Operation interface {
	Describe(*Project) string
	Do(context.Context, *Action, *Env) error
}

type OperationFunc func(context.Context, *Action, *Env) error

func (of OperationFunc) Do(ctx context.Context, a *Action, env *Env) error {
	return of(ctx, a, env)
}

type ActionBuilder interface {
	BuildAction(prj *Project, premises, results []*Goal) (*Action, error)
}

var Implicit implicit

type implicit struct{}

func (implicit) BuildAction(prj *Project, premises, results []*Goal) (*Action, error) {
	a := prj.NewAction(premises, results, nil)
	return a, nil
}

type badOp struct {
	op  Operation
	err error
}

var _ Operation = badOp{}

func (bop badOp) Describe(prj *Project) string {
	if bop.op == nil {
		return fmt.Sprintf("bad operation:%s", bop.err)
	}
	return fmt.Sprintf("bad operation:%s:%s", bop.op.Describe(prj), bop.err)
}

func (bop badOp) Do(_ context.Context, a *Action, _ *Env) error {
	return fmt.Errorf("called %s", bop.Describe(a.Project()))
}
