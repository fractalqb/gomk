package gomkore

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

type TracerCommon interface {
	Debug(t *Trace, msg string, args ...any)
	Info(t *Trace, msg string, args ...any)
	Warn(t *Trace, msg string, args ...any)

	StartProject(t *Trace, p *Project, activity string)
	DoneProject(t *Trace, p *Project, activity string, dt time.Duration)
}

type Tracer interface {
	BuildTracer
	CleanTracer
}

type TraceLog int

var DefaultTraceLog TraceLog = TraceWarn

const (
	TraceWarn TraceLog = (1 << iota)
	TraceInfo
	TraceDebug
)

type Trace struct {
	root *traceRoot
	up   *Trace
	obj  any
	id   uint64
}

func NewTrace(ctx context.Context, t Tracer) *Trace {
	root := &traceRoot{ctx: ctx, tr: t}
	return &Trace{root: root}
}

func (t *Trace) Ctx() context.Context { return t.root.ctx }

func (t *Trace) Debug(msg string, args ...any) { t.root.tr.Debug(t, msg, args...) }
func (t *Trace) Info(msg string, args ...any)  { t.root.tr.Info(t, msg, args...) }
func (t *Trace) Warn(msg string, args ...any)  { t.root.tr.Warn(t, msg, args...) }

func (t *Trace) startProject(p *Project, activity string) {
	t.root.prj = p
	t.root.tr.StartProject(t, p, activity)
}

func (t *Trace) doneProject(p *Project, activity string, dt time.Duration) {
	t.root.tr.DoneProject(t, p, activity, dt)
	t.root.prj = nil
}

func (t *Trace) runAction(a *Action) {
	t.root.tr.RunAction(t, a)
}

func (t *Trace) runImplicitAction(a *Action) {
	t.root.tr.RunImplicitAction(t, a)
}

func (t *Trace) scheduleResTimeZero(a *Action, res *Goal) {
	t.root.tr.ScheduleResTimeZero(t, a, res)
}

func (t *Trace) scheduleNotPremises(a *Action, res *Goal) {
	t.root.tr.ScheduleNotPremises(t, a, res)
}

func (t *Trace) schedulePreTimeZero(a *Action, res, pre *Goal) {
	t.root.tr.SchedulePreTimeZero(t, a, res, pre)
}

func (t *Trace) scheduleOutdated(a *Action, res, pre *Goal) {
	t.root.tr.ScheduleOutdated(t, a, res, pre)
}

func (t *Trace) checkGoal(g *Goal) {
	t.root.tr.CheckGoal(t, g)
}

func (t *Trace) goalUpToDate(g *Goal) {
	t.root.tr.GoalUpToDate(t, g)
}

func (t *Trace) goalNeedsActions(g *Goal, n int) {
	t.root.tr.GoalNeedsActions(t, g, n)
}

func (t *Trace) removeArtefact(g *Goal) {
	t.root.tr.RemoveArtefact(t, g)
}

func (t *Trace) Build() BuildID {
	if t.root == nil {
		return 0
	}
	return t.root.prj.Build()
}

func (t *Trace) TopID() uint64 { return t.id }

func (t *Trace) TopTag() string {
	switch t.obj.(type) {
	case *Goal:
		return fmt.Sprintf("[%d]", t.id)
	case *Action:
		return fmt.Sprintf("(%d)", t.id)
	case *Project:
		return fmt.Sprintf("{%d}", t.id)
	case nil:
		return ""
	}
	return fmt.Sprintf("!%T!", t.obj)
}

func (t *Trace) Path() string {
	var sb strings.Builder
	sb.WriteByte('<')
	for ; t != nil; t = t.up {
		sb.WriteString(t.TopTag())
	}
	sb.WriteByte('>')
	return sb.String()
}

func (t *Trace) String() string {
	if t.root.prj == nil {
		return t.Path()
	}
	return fmt.Sprintf("%d@%s", t.root.prj.Build(), t.Path())
}

func (t *Trace) pushProject(p *Project) *Trace {
	return &Trace{
		root: t.root,
		up:   t,
		obj:  p,
		id:   t.root.idSeq.Add(1),
	}
}

func (t *Trace) pushGoal(g *Goal) *Trace {
	return &Trace{
		root: t.root,
		up:   t,
		obj:  g,
		id:   t.root.idSeq.Add(1),
	}
}

type traceRoot struct {
	ctx   context.Context
	tr    Tracer
	prj   *Project
	idSeq atomic.Uint64
}
