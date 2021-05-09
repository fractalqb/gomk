package gomk

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

type Tasks map[string]*taskdef

func (ts Tasks) Def(name string, do func(*WDir), before ...string) {
	if _, ok := ts[name]; ok {
		panic(fmt.Errorf("task '%s' redefined", name))
	}
	ts[name] = &taskdef{
		do:     do,
		before: before,
	}
}

func (ts Tasks) Before(name string) []string {
	td := ts[name]
	if td == nil || len(td.before) == 0 {
		return nil
	}
	return td.before
}

func (ts Tasks) List() []string {
	res := make([]string, 0, len(ts))
	for name := range ts {
		res = append(res, name)
	}
	sort.Strings(res)
	return res
}

func (ts Tasks) Fprint(wr io.Writer, prefix string) {
	twr, ok := wr.(*tabwriter.Writer)
	if !ok {
		twr = tabwriter.NewWriter(wr,
			0, 0,
			1, ' ',
			0,
		)
	}
	for _, t := range ts.List() {
		fmt.Fprintf(twr, "%s%s\t<- %s\n", prefix, t, ts.Before(t))
	}
	twr.Flush()
}

func (ts Tasks) Run(task string, dir *WDir) {
	t := ts[task]
	if t == nil {
		panic(fmt.Errorf("no task '%s'", task))
	}
	if t.done {
		return
	}
	for _, b := range t.before {
		ts.Run(b, dir)
	}
	if t.do != nil {
		t.do(dir)
	}
}

type taskdef struct {
	do     func(*WDir)
	before []string
	done   bool
}
