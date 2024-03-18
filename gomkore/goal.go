package gomkore

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/bits-and-blooms/bitset"
)

// Artefact represents the tangible outcome of a [Goal] being reached. A special
// case is the [Abstract] artefact.
type Artefact interface {
	Key() any

	// Name returns the name of the artefact that must be unique in the Project.
	Name(in *Project) string

	// StateAs returns the time at which the artefact reached its current state.
	// If this cannot be provided, the zero Time is returned.
	StateAt(in *Project) (time.Time, error)
}

type RemovableArtefact interface {
	Artefact
	Exists(in *Project) (bool, error)
	Remove(in *Project) error
}

type Abstract string

var _ Artefact = Abstract("")

func (a Abstract) Key() any { return a }

func (a Abstract) Name(*Project) string { return string(a) }

func (a Abstract) StateAt(prj *Project) (t time.Time, err error) {
	g, err := prj.Goal(a)
	if err != nil {
		return time.Time{}, err
	}
	for _, act := range g.ResultOf() {
		if act.Op != nil {
			return time.Time{}, nil
		}
	}
	for _, act := range g.ResultOf() {
		for _, pre := range act.Premises() {
			if pt, err := pre.Artefact.StateAt(prj); err != nil {
				return pt, err
			} else if pt.After(t) {
				t = pt
			}
		}
	}
	return t, nil
}

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

type GoalFactory interface {
	Goals(in *Project) ([]*Goal, error)
}

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
	Artefact   Artefact
	Removable  bool

	prj       *Project
	resultOf  []*Action
	premiseOf []*Action

	sync.Mutex
	lastBID BuildID
}

func (g *Goal) Project() *Project { return g.prj }

func (g *Goal) Name() string { return g.Artefact.Name(g.Project()) }

// ResultOf returns the actions that result in this goal.
func (g *Goal) ResultOf() []*Action { return g.resultOf }

// PreAction returns [Goal.ResultOf]()[i]
func (g *Goal) PreAction(i int) *Action { return g.resultOf[i] }

// PremiseOf returns the actions on which g depends.
func (g *Goal) PremiseOf() []*Action { return g.premiseOf }

// PostAction returns [Goal.PremiseOf]()[i]
func (g *Goal) PostAction(i int) *Action { return g.premiseOf[i] }

func (g *Goal) IsAbstract() bool {
	_, ok := g.Artefact.(Abstract)
	return ok
}

// Requires 'involved' to really be involved
func (g *Goal) UpdateConsistency(involved *Goal) error {
	// TODO This has to be carefully aligned with the builder
	//
	// ~ A goal that has
	// been partially build as "involved" only calls missing actions. For the
	// time being be pessimistic!

	if involved == g {
		return nil
	}
	switch len(g.ResultOf()) {
	case 0:
		return nil
	case 1:
		if len(involved.ResultOf()) <= 1 {
			return nil
		}
	}
	if len(g.ResultOf()) != len(involved.ResultOf()) {
		return fmt.Errorf("different number of actions for goal %s and involved goal %s",
			g,
			involved,
		)
	}
	if involved.UpdateMode.Ordered() {
		if !g.UpdateMode.Ordered() {
			return fmt.Errorf("update conflict of unordered goal %s with ordered involved goal %s",
				g,
				involved,
			)
		}
		for i, ga := range g.ResultOf() {
			if ga != involved.PreAction(i) {
				return fmt.Errorf("different actions for goal %s and involved goal %s",
					g,
					involved,
				)
			}
		}
		return nil
	}
	if g.UpdateMode.Ordered() {
		return fmt.Errorf("update conflict of ordered goal %s with unordered involved goal %s",
			g,
			involved,
		)
	}
	for _, ga := range g.ResultOf() {
		if slices.Index(involved.ResultOf(), ga) < 0 {
			return fmt.Errorf("different actions for goal %s and involved goal %s",
				g,
				involved,
			)
		}
	}
	return nil
}

func (g *Goal) String() string {
	tn := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()
	an := g.Name()
	return fmt.Sprintf("%s:%s", an, tn)
}

// CheckPreTimes check if g needs to be updated according to the timestamps of
// all of its premises.
func (g *Goal) CheckPreTimes(tr *Trace) (chgs []int, err error) {
	// TODO Consistency for concurrent builds
	gaTS, err := g.Artefact.StateAt(g.Project())
	if err != nil {
		return nil, err
	}
	for actIdx, act := range g.ResultOf() {
		if gaTS.IsZero() {
			tr.scheduleResTimeZero(act, g)
			chgs = append(chgs, actIdx)
			continue
		} else if len(act.Premises()) == 0 {
			tr.scheduleNotPremises(act, g)
			chgs = append(chgs, actIdx)
			continue
		}
	PREMISE_LOOP:
		for _, pre := range act.Premises() {
			preTS, err := pre.Artefact.StateAt(g.Project())
			if err != nil {
				return nil, err
			}
			switch {
			case preTS.IsZero():
				tr.schedulePreTimeZero(act, g, pre)
				chgs = append(chgs, actIdx)
				break PREMISE_LOOP
			case gaTS.Before(preTS):
				tr.scheduleOutdated(act, g, pre)
				chgs = append(chgs, actIdx)
				break PREMISE_LOOP
			}
		}
	}
	return chgs, nil
}

// LockBuild locks g once for the current build of g's project. If g was already
// locked for the build 0 is returned.
func (g *Goal) LockBuild() BuildID {
	g.Mutex.Lock()
	if plb := g.Project().lastBuild; g.lastBID < plb {
		g.lastBID = plb
		return plb
	}
	g.Mutex.Unlock()
	return 0
}

func (g *Goal) LockPreActions(gid uintptr) {
	todo := len(g.ResultOf())
	locked := bitset.New(uint(todo))

	var (
		i  uint = math.MaxUint
		ok bool
	)
	for todo > 0 {
		if i, ok = locked.NextClear(i + 1); !ok {
			i, ok = locked.NextClear(0)
			if !ok {
				panic("no next to lock but todo > 0")
			}
		}
		blockGID := g.resultOf[i].tryLock(gid)
		if blockGID > gid { // I lost => restart
			for j, ok := locked.NextSet(0); ok; j, ok = locked.NextSet(j + 1) {
				g.resultOf[j].unlock()
			}
			locked.ClearAll()
			todo = len(g.ResultOf())
			// Sleep for short to not stay in the winner's way
			time.Sleep(time.Millisecond) // TODO reasonable?
		} else {
			locked.Set(i)
			todo--
		}
	}
}

func (g *Goal) UnlockPreActions() {
	for _, act := range g.ResultOf() {
		act.unlock()
	}
}
