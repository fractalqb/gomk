package gomk

import (
	"context"
	"errors"
	"fmt"
)

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
	if env == nil {
		env = DefaultEnv()
	}
	if a.Op == nil {
		return nil
	}
	return a.Op.Do(ctx, a, env)
}

func (a *Action) Valid() error {
	if bop, ok := a.Op.(badOp); ok {
		return errors.New(bop.Describe(a.Project()))
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

type badOp struct {
	op  Operation
	err error
}

func (bop badOp) Describe(prj *Project) string {
	if bop.op == nil {
		return fmt.Sprintf("bad operation:%s", bop.err)
	}
	return fmt.Sprintf("bad operation:%s:%s", bop.op.Describe(prj), bop.err)
}

func (bop badOp) Do(_ context.Context, a *Action, _ *Env) error {
	return fmt.Errorf("called %s", bop.Describe(a.Project()))
}
