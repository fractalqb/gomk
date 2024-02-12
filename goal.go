package gomk

import (
	"errors"
	"fmt"
	"hash"
	"reflect"
	"time"
)

type UpdateMode uint

const (
	// All actions must be run to reach the goal.
	UpdAllActions UpdateMode = 0
	// All actions with changed state must be run to reach the goal.
	UpdSomeActions UpdateMode = 1
	// Only one of the actions wit changed state has to be run to reach the
	// goal.
	UpdAnyAction UpdateMode = 2
	// Only one action must have changed state. Then the goal is reached by
	// running that action.
	UpdOneAction UpdateMode = 3

	UpdUnordered UpdateMode = 4

	updActions UpdateMode = 3
)

func (m UpdateMode) Actions() UpdateMode { return m & updActions }
func (m UpdateMode) Ordered() bool       { return (m & UpdUnordered) == 0 }

// A Goal is something you want to achieve in your [Project]. Each goal is
// associated with an [Artefact] – generally something tangible that is
// considered available and up-to-date when the goal is achieved. A special case
// is the [Abstract] artefact that simply provides a name for abstract goals.
// Goals can be achieved through actions. The goal is then a result of each
// [Action]. On the other hand, a goal can also be the premise for one or more
// actions. Such dependent actions may not be carried out before the goal is
// reached.
type Goal struct {
	UpdateMode UpdateMode
	ResultOf   []*Action // Actions that result in this goal.
	PremiseOf  []*Action // Dependent actions of this goal.
	Artefact   Artefact

	prj       *Project
	lastBuild int64
	stateAt   time.Time
	stateHash []byte
}

func (g *Goal) By(a ActionBuilder, premises ...*Goal) *Goal {
	_, err := a.BuildAction(g.Project(), premises, []*Goal{g})
	if err != nil {
		g.Project().NewAction(premises, []*Goal{g}, badOp{err: err})
		return g
	}
	return g
}

func (g *Goal) ImpliedBy(premises ...*Goal) *Goal {
	g.Project().NewAction(premises, []*Goal{g}, nil)
	return g
}

func (g *Goal) Project() *Project { return g.prj }

func (g *Goal) Name() string { return g.Artefact.Name(g.Project()) }

func (g *Goal) IsAbstract() bool {
	_, ok := g.Artefact.(Abstract)
	return ok
}

// Valid checks all premise and result actions of g using [Action.Valid] and
// returns g along with all collected errors, if any.
func (g *Goal) Valid() (*Goal, error) {
	var errs []error
	for _, a := range g.ResultOf {
		if err := a.Valid(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, a := range g.PremiseOf {
		if err := a.Valid(); err != nil {
			errs = append(errs, err)
		}
	}
	switch len(errs) {
	case 0:
		return g, nil
	case 1:
		return g, errs[1]
	}
	return g, errors.Join(errs...)
}

func (g *Goal) String() string {
	tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
	an := g.Name()
	return fmt.Sprintf("[%s]%s", an, tn)
}

type Artefact interface {
	Name(in *Project) string
	// StateAs returns the time at which the artefact reached its current state.
	// If this cannot be provided, the zero Time is returned.
	StateAt() time.Time
}

type Abstract string

var _ Artefact = Abstract("")

func (a Abstract) Name(*Project) string { return string(a) }

func (a Abstract) StateAt() time.Time { return time.Time{} }

type HashableArtefact interface {
	Artefact
	StateHash(hash.Hash) error
}

func HashArtefcat(a Artefact, h hash.Hash) (bool, error) {
	if ha, ok := a.(HashableArtefact); ok {
		return true, ha.StateHash(h)
	}
	return false, nil
}
