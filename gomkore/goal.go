package gomkore

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

type UpdateMode uint

const (
	// All actions must be run to reach the goal.
	UpdAllActions UpdateMode = 0

	// All actions with changed state must be run to reach the goal.
	UpdSomeActions UpdateMode = 1

	// Only one of the actions with changed state has to be run to reach the
	// goal.
	UpdAnyAction UpdateMode = 2

	// Only one action must have changed state. Then the goal is reached by
	// running that action.
	UpdOneAction UpdateMode = 3

	// An unordered update mode allows actions of the current goal to be run in
	// any order or even concurrently. Otherwise, the actions must be run one
	// after the other in the specified order.
	UpdUnordered UpdateMode = 4

	updActions UpdateMode = 3
)

func (m UpdateMode) Actions() UpdateMode { return m & updActions }
func (m UpdateMode) Ordered() bool       { return (m & UpdUnordered) == 0 }

// A Goal is something you want to achieve in your [Project]. Each goal is
// associated with an [Artefact] – generally something tangible that is
// considered available and up-to-date when the goal is achieved. A special case
// is the [Abstract] artefact that simply provides a name for abstract goals.
// Abstract goals do not deliver tangible results.
//
// Goals can be achieved through actions ([Action]). A goal can be the result of
// several actions at the same time. It then depends on the target's
// [UpdateMode] whether and how the actions contribute to the target. On the
// other hand, a goal can also be the premise for one or more actions. Such
// dependent actions should not be carried out before the goal is reached.
type Goal struct {
	UpdateMode UpdateMode
	ResultOf   []*Action // Actions that result in this goal.
	PremiseOf  []*Action // Dependent actions of this goal.
	Artefact   Artefact

	sync.Mutex

	prj       *Project
	lastBuid  BuildID
	buildInfo any // TODO is it necessary?
}

func (g *Goal) Project() *Project { return g.prj }

func (g *Goal) Name() string { return g.Artefact.Name(g.Project()) }

func (g *Goal) IsAbstract() bool {
	_, ok := g.Artefact.(Abstract)
	return ok
}

// Requires 'involved' to really be involved
func (g *Goal) UpdateConsistency(involved *Goal) error {
	if involved == g {
		return nil
	}
	switch len(g.ResultOf) {
	case 0:
		return nil
	case 1:
		if len(involved.ResultOf) <= 1 {
			return nil
		}
	}
	return fmt.Errorf("update conflict of goal %s with involved goal %s",
		g,
		involved,
	)
	// TODO This has to be carefully aligned with the builder:
	// ~ A goal that has been partially build as "involved" only calls missing
	// actions
	//
	// switch len(g.ResultOf) {
	// case 0:
	// 	return nil
	// case 1:
	// 	if len(involved.ResultOf) <= 1 {
	// 		return nil // Even if involved is not really involved => no conflict
	// 	}
	// 	switch involved.UpdateMode.Actions() {
	// 	case UpdAllActions, UpdSomeActions:
	// 		// If g's action is 1st (ordred?) of involved => could be done
	// 		return fmt.Errorf("involved goal %s partially updated by %s",
	// 			involved,
	// 			g,
	// 		)
	// 	case UpdAnyAction, UpdOneAction:
	// 		return nil
	// 	}
	// 	panic("unreachable code")
	// }
	// if !involved.UpdateMode.Ordered() {
	// 	switch involved.UpdateMode.Actions() {
	// 	case UpdAllActions:
	// 		// TODO Assert g ordered & same list of actions
	// 	case UpdSomeActions:
	// 		return nil
	// 	case UpdAnyAction, UpdOneAction:
	// 		// TODO Assert g hase exactly 1 same action
	// 	}
	// 	panic("unreachable code")
	// }
	// if !g.UpdateMode.Ordered() {
	// 	if len(g.ResultOf) == 1 { // (v.s.) => len(involved.ResultOf) > 1
	// 		switch involved.UpdateMode.Actions() {
	// 		}
	// 	}
	// 	return errors.New("unordered involves ordered")
	// }
	// return errors.New("NYI: Goal.UpdateConsistency()")
}

func (g *Goal) String() string {
	tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
	an := g.Name()
	return fmt.Sprintf("[%s]%s", an, tn)
}

func (g *Goal) LockBuild(info func() any) (BuildID, any) {
	g.Lock()
	if plb := g.Project().lastBuild; g.lastBuid < plb {
		g.lastBuid = plb
		g.buildInfo = info()
		return plb, g.buildInfo
	}
	g.Unlock()
	return 0, nil
}

// BuildInfo must only be called by the goroutine that holds the lock.
func (g *Goal) BuildInfo() any { return g.buildInfo }

// Artefact represents the tangible outcome of a [Goal] being reached. A special
// case is the [Abstract] artefact.
type Artefact interface {
	// Name returns the name of the artefact that must be unique in the Project.
	Name(in *Project) string

	// StateAs returns the time at which the artefact reached its current state.
	// If this cannot be provided, the zero Time is returned.
	StateAt(in *Project) time.Time
}

type Abstract string

var _ Artefact = Abstract("")

func (a Abstract) Name(*Project) string { return string(a) }

func (a Abstract) StateAt(*Project) time.Time { return time.Time{} }
