package gomk

import (
	"testing"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type TestTracer struct{ t *testing.T }

var _ gomkore.Tracer = TestTracer{}

func (tr TestTracer) Debug(t *gomkore.Trace, msg string, args ...any) {
	tr.t.Logf("gomk-DEBUG: "+msg, args...)
}

func (tr TestTracer) Info(t *gomkore.Trace, msg string, args ...any) {
	tr.t.Logf("gomk-INFO: "+msg, args...)
}

func (tr TestTracer) Warn(t *gomkore.Trace, msg string, args ...any) {
	tr.t.Logf("gomk-WARN: "+msg, args...)
}

func (tr TestTracer) StartProject(t *gomkore.Trace, p *gomkore.Project, activity string) {
	tr.t.Logf("gomk-StartProject: %s %s", p, activity)
}

func (tr TestTracer) DoneProject(t *gomkore.Trace, p *gomkore.Project, activity string, dt time.Duration) {
	tr.t.Logf("gomk-DoneProject: %s %s %s", p, activity, dt)
}

func (tr TestTracer) SetupActionEnv(t *gomkore.Trace, env *gomkore.Env) (*gomkore.Env, error) {
	return env, nil
}

func (tr TestTracer) CloseActionEnv(t *gomkore.Trace, env *gomkore.Env) error { return nil }

func (tr TestTracer) RunAction(_ *gomkore.Trace, a *gomkore.Action) {
	tr.t.Logf("gomk-RunAction: %s", a)
}

func (tr TestTracer) RunImplicitAction(_ *gomkore.Trace, a *gomkore.Action) {
	tr.t.Logf("gomk-RunImplicitAction: %s", a)
}

func (tr TestTracer) ScheduleResTimeZero(t *gomkore.Trace, a *gomkore.Action, res *gomkore.Goal) {
	tr.t.Logf("gomk-ScheduleResTimeZero: %s:> %s", a, res)
}

func (tr TestTracer) ScheduleNotPremises(t *gomkore.Trace, a *gomkore.Action, res *gomkore.Goal) {
	tr.t.Logf("gomk-ScheduleNotPremises: %s:> %s", a, res)
}

func (tr TestTracer) SchedulePreTimeZero(t *gomkore.Trace, a *gomkore.Action, res, pre *gomkore.Goal) {
	tr.t.Logf("gomk-SchedulePreTimeZero: %s: %s > %s", a, pre, res)
}

func (tr TestTracer) ScheduleOutdated(t *gomkore.Trace, a *gomkore.Action, res, pre *gomkore.Goal) {
	tr.t.Logf("gomk-ScheduleOutdated: %s: %s > %s", a, pre, res)
}

func (tr TestTracer) CheckGoal(t *gomkore.Trace, g *gomkore.Goal) {
	tr.t.Logf("gomk-CheckGoal: %s", g)
}

func (tr TestTracer) GoalUpToDate(t *gomkore.Trace, g *gomkore.Goal) {
	tr.t.Logf("gomk-GoalUpToDate: %s", g)
}

func (tr TestTracer) GoalNeedsActions(t *gomkore.Trace, g *gomkore.Goal, n int) {
	tr.t.Logf("gomk-GoalNeedsActions: %s %d", g, n)
}

func (tr TestTracer) RemoveArtefact(t *gomkore.Trace, g *gomkore.Goal) {
	tr.t.Logf("gomk-RemoveArtefact: %s", g)
}
