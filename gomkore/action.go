package gomkore

import (
	"hash"
	"sync/atomic"
)

type Operation interface {
	// The hints are optional
	Describe(actionHint *Action, envHint *Env) string
	Do(tr *Trace, a *Action, env *Env) error
	WriteHash(h hash.Hash, a *Action, env *Env) (bool, error)
}

// An Action is something you can do in your [Project] to achieve at least one
// [Goal]. The actual implementation of the action is an [Operation]. An action
// without an operation is an "implicit" action, i.e. if all its premises are
// true, all results of the action are implicitly given.
type Action struct {
	Op          Operation
	IgnoreError bool

	prj      *Project
	premises []*Goal
	results  []*Goal

	lockGID uintptr
	lastBID BuildID
}

func (a *Action) Project() *Project   { return a.prj }
func (a *Action) Premises() []*Goal   { return a.premises }
func (a *Action) Premise(i int) *Goal { return a.premises[i] }
func (a *Action) Results() []*Goal    { return a.results }
func (a *Action) Result(i int) *Goal  { return a.results[i] }

func (a *Action) LastBuild() BuildID { return a.lastBID }

// Must not run concurrently, see [Goal.LockPreActions] and [tryLock]
func (a *Action) Run(tr *Trace, env *Env) (BuildID, error) {
	if tr.Build() <= a.lastBID {
		return a.lastBID, nil
	}
	a.lastBID = tr.Build()
	if env == nil {
		env = DefaultEnv(tr)
	}
	if a.Op == nil {
		tr.runImplicitAction(a)
		return 0, nil
	}
	var err error
	env, err = tr.setupActionEnv(env)
	if err != nil {
		return 0, err
	}
	defer tr.closeActionEnv(env)
	tr.runAction(a)
	err = a.Op.Do(tr, a, env)
	switch {
	case err == nil:
		return 0, nil
	case a.IgnoreError:
		tr.Warn("ignoring `action` `error`",
			`action`, a,
			`error`, err,
		)
		return 0, nil
	}
	return 0, err
}

func (a *Action) String() string {
	switch {
	case a == nil:
		return "<nil:action>/" + a.Project().Name(nil)
	case a.Op == nil:
		return "<implicit>/" + a.Project().Name(nil)
	}
	return a.Op.Describe(a, nil)
}

func (a *Action) WriteHash(h hash.Hash, env *Env) (bool, error) {
	if a.Op == nil {
		return false, nil
	}
	return a.Op.WriteHash(h, a, env)
}

func (a *Action) tryLock(byGID uintptr) (blockingGID uintptr) {
	if atomic.CompareAndSwapUintptr(&a.lockGID, 0, byGID) {
		return 0
	}
	return atomic.LoadUintptr(&a.lockGID)
}

func (a *Action) unlock() { atomic.StoreUintptr(&a.lockGID, 0) }

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
