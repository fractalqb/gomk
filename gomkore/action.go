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
	Op          Operation
	IgnoreError bool
	premises    []*Goal
	results     []*Goal

	prj *Project
}

func (a *Action) Project() *Project   { return a.prj }
func (a *Action) Premises() []*Goal   { return a.premises }
func (a *Action) Premise(i int) *Goal { return a.premises[i] }
func (a *Action) Results() []*Goal    { return a.results }
func (a *Action) Result(i int) *Goal  { return a.results[i] }

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
	err := a.Op.Do(ctx, a, env)
	switch {
	case err == nil:
		return nil
	case a.IgnoreError:
		env.Log.Warn("ignoring `action` `error`",
			`action`, a.String(),
			`error`, err,
		)
		return nil
	}
	return err
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

func updateConsistency(results []*Goal) error {
	for _, g := range results {
		for _, i := range results {
			if err := g.UpdateConsistency(i); err != nil {
				return err
			}
		}
	}
	return nil
}
