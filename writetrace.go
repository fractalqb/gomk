package gomk

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/sllm/v3"
)

type TraceLevel int

var DefaultTraceLevel TraceLevel = TraceLeast

const (
	TraceImportant TraceLevel = (1 << iota)
	TraceNormal
	TraceDetails

	TraceNothing = 0
	TraceLeast   = TraceImportant
	TraceMedium  = TraceLeast | TraceNormal
	TraceMost    = TraceMedium | TraceDetails
)

func (cfg TraceLevel) Traces(l TraceLevel) bool { return cfg&l == l }

type WriteTracer struct {
	W   io.Writer
	Log TraceLevel
}

func NewDefaultTracer() *WriteTracer {
	return &WriteTracer{W: os.Stderr, Log: TraceImportant}
}

const WriteTraceLevelFlagDoc = `Set trace level: l/least; m/medium; M/most`

func (tr *WriteTracer) ParseLevelFlag(f string) error {
	switch f {
	case "":
		return nil
	case "off":
		tr.Log = TraceNothing
	case "least", "l":
		tr.Log = TraceLeast
	case "medium", "m":
		tr.Log = TraceMedium
	case "most", "M":
		tr.Log = TraceMost
	default:
		return fmt.Errorf("write tracer: illegal log flag '%s'", f)
	}
	return nil
}

func (tr WriteTracer) Debug(t *gomkore.Trace, msg string, args ...any) {
	if tr.Log.Traces(TraceDetails) {
		fmt.Fprintf(tr.W, "%d@%s\t  DEBUG ", t.Build(), t.TopTag())
		sllm.Fprint(tr.W, msg, sllmArgs(args).append)
		fmt.Fprintln(tr.W)
	}
}

func (tr WriteTracer) Info(t *gomkore.Trace, msg string, args ...any) {
	if tr.Log.Traces(TraceNormal) {
		fmt.Fprintf(tr.W, "%d@%s\t  INFO  ", t.Build(), t.TopTag())
		sllm.Fprint(tr.W, msg, sllmArgs(args).append)
		fmt.Fprintln(tr.W)
	}
}

func (tr WriteTracer) Warn(t *gomkore.Trace, msg string, args ...any) {
	if tr.Log.Traces(TraceImportant) {
		fmt.Fprintf(tr.W, "%d@%s\t  WARN  ", t.Build(), t.TopTag())
		sllm.Fprint(tr.W, msg, sllmArgs(args).append)
		fmt.Fprintln(tr.W)
	}
}

func (tr WriteTracer) StartProject(t *gomkore.Trace, p *gomkore.Project, activity string) {
	if tr.Log != 0 {
		fmt.Fprintf(tr.W, "%d@%s\t{ %s project '%s' in %s\n",
			t.Build(),
			t.TopTag(),
			activity,
			p,
			p.Dir,
		)
	}
}

func (tr WriteTracer) DoneProject(t *gomkore.Trace, p *gomkore.Project, activity string, dt time.Duration) {
	if tr.Log != 0 {
		fmt.Fprintf(tr.W, "%d@%s\t} %s project '%s' took %s\n",
			t.Build(),
			t.TopTag(),
			activity,
			p,
			dt,
		)
	}
}

func (tr WriteTracer) SetupActionEnv(t *gomkore.Trace, env *gomkore.Env) (*gomkore.Env, error) {
	if env.Out == nil && env.Err == nil {
		return env, nil
	}
	e := env.Sub()
	var pre bytes.Buffer
	if e.Out != nil {
		fmt.Fprintf(&pre, "%d@%s Out: ", t.Build(), t.TopTag())
		e.Out = newPrefixWriter(e.Out, bytes.Clone(pre.Bytes()))
	}
	if e.Err != nil {
		pre.Reset()
		fmt.Fprintf(&pre, "%d@%s Err: ", t.Build(), t.TopTag())
		e.Err = newPrefixWriter(e.Err, pre.Bytes())
	}
	return e, nil
}

func (tr WriteTracer) CloseActionEnv(t *gomkore.Trace, env *gomkore.Env) error { return nil }

func (tr WriteTracer) RunAction(t *gomkore.Trace, a *gomkore.Action) {
	if tr.Log.Traces(TraceImportant) {
		fmt.Fprintf(tr.W, "%d@%s\t  run action (%s)\n", t.Build(), t.TopTag(), a)
	}
}

func (tr WriteTracer) RunImplicitAction(t *gomkore.Trace, _ *gomkore.Action) {
	if tr.Log.Traces(TraceDetails) {
		fmt.Fprintf(tr.W, "%d@%s\t  implicit action\n", t.Build(), t.TopTag())
	}
}

func (tr WriteTracer) ScheduleResTimeZero(t *gomkore.Trace, a *gomkore.Action, res *gomkore.Goal) {
	if tr.Log.Traces(TraceNormal) {
		fmt.Fprintf(tr.W, "%d@%s\t  schedule (%s) for result [%s] without state time\n",
			t.Build(),
			t.TopTag(),
			a,
			res,
		)
	}
}

func (tr WriteTracer) ScheduleNotPremises(t *gomkore.Trace, a *gomkore.Action, res *gomkore.Goal) {
	if tr.Log.Traces(TraceNormal) {
		fmt.Fprintf(tr.W, "%d@%s\t  schedule (%s) without premise for result [%s]\n",
			t.Build(),
			t.TopTag(),
			a,
			res,
		)
	}
}

func (tr WriteTracer) SchedulePreTimeZero(t *gomkore.Trace, a *gomkore.Action, res, pre *gomkore.Goal) {
	if tr.Log.Traces(TraceNormal) {
		fmt.Fprintf(tr.W, "%d@%s\t  schedule (%s) for result [%s], premise [%s] has no state time\n",
			t.Build(),
			t.TopTag(),
			a,
			res,
			pre,
		)
	}
}

func (tr WriteTracer) ScheduleOutdated(t *gomkore.Trace, a *gomkore.Action, res, pre *gomkore.Goal) {
	if tr.Log.Traces(TraceNormal) {
		fmt.Fprintf(tr.W, "%d@%s\t  schedule (%s) for result [%s], premise [%s] is newer\n",
			t.Build(),
			t.TopTag(),
			a,
			res,
			pre,
		)
	}
}

func (tr WriteTracer) CheckGoal(t *gomkore.Trace, g *gomkore.Goal) {
	if tr.Log.Traces(TraceImportant) {
		fmt.Fprintf(tr.W, "%d@%s\t? [%s] %s\n",
			t.Build(),
			t.TopTag(),
			g,
			t.Path(),
		)
	}
}

func (tr WriteTracer) GoalUpToDate(t *gomkore.Trace, g *gomkore.Goal) {
	if tr.Log.Traces(TraceImportant) {
		fmt.Fprintf(tr.W, "%d@%s\t. [%s] is up-to-date\n",
			t.Build(),
			t.TopTag(),
			g,
		)
	}
}

func (tr WriteTracer) GoalNeedsActions(t *gomkore.Trace, g *gomkore.Goal, n int) {
	if tr.Log.Traces(TraceImportant) {
		fmt.Fprintf(tr.W, "%d@%s\t! [%s] needs %d actions\n",
			t.Build(),
			t.TopTag(),
			g,
			n,
		)
	}
}

func (tr WriteTracer) RemoveArtefact(t *gomkore.Trace, g *gomkore.Goal) {
	if tr.Log.Traces(TraceImportant) {
		fmt.Fprintf(tr.W, "%d@%s\t! remove artefact [%s]\n",
			t.Build(),
			t.TopTag(),
			g,
		)
	}
}

type sllmArgs []any

func (as sllmArgs) append(buf []byte, _ int, n string) ([]byte, error) {
	for len(as) > 0 {
		switch k := as[0].(type) {
		case string:
			if len(as) == 1 {
				return buf, fmt.Errorf("no value for key '%s'", n)
			}
			if k == n {
				return sllm.AppendArg(buf, as[1]), nil
			}
			as = as[2:]
		case slog.Attr:
			if k.Key == n {
				return sllm.AppendArg(buf, k.Value), nil
			}
			as = as[1:]
		default:
			return buf, fmt.Errorf("illegal key type %T", k)
		}
	}
	return buf, fmt.Errorf("no key '%s", n)
}
