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

func newAction(ps, rs []*Goal, op Operation) *Action {
	prj, err := CheckProject(nil, ps, rs)
	if err != nil {
		op = badOp{op: op, err: err}
	} else if prj == nil {
		op = badOp{op: op, err: errors.New("no project")}
	}
	a := &Action{
		Premises: ps,
		Results:  rs,
		Op:       op,
		prj:      prj,
	}
	for _, p := range ps {
		p.PremiseOf = append(p.PremiseOf, a)
	}
	for _, r := range rs {
		r.ResultOf = append(r.ResultOf, a)
	}
	return a
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
	NewAction(prj *Project, premises, results []*Goal) (*Action, error)
}

func CheckProject(prj *Project, premises, results []*Goal) (*Project, error) {
	for _, g := range premises {
		if prj == nil {
			prj = g.Project()
		} else if p := g.Project(); p != prj {
			return nil, fmt.Errorf("premise '%s' not in project '%s'",
				p.String(),
				prj.String(),
			)
		}
	}
	for _, g := range results {
		if prj == nil {
			prj = g.Project()
		} else if p := g.Project(); p != prj {
			return nil, fmt.Errorf("result '%s' not in project '%s'",
				p.String(),
				prj.String(),
			)
		}
	}
	return prj, nil
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
