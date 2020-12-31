package gomk

import (
	"fmt"
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
	t.do(dir)
}

type taskdef struct {
	do     func(*WDir)
	before []string
	done   bool
}
