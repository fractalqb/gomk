package gomk

import (
	"context"
	"fmt"
)

func Build(p *Project, task string) error {
	env := TaskEnv{
		Ctx:   context.Background(),
		Trace: LogTracer,
	}
	done := make(map[string]error)
	return bld(env, done, p, task)
}

func bld(env TaskEnv, done map[string]error, p *Project, task string) error {
	if err, ok := done[task]; ok {
		return err
	}
	t := p.Task(task)
	if t == nil {
		return fmt.Errorf("no task '%s' in project '%s'", task, p.Name())
	}
	for _, b := range t.DependsOn() {
		if err := bld(env, done, p, b); err != nil {
			return err
		}
	}
	err := t.Run(env)
	done[task] = err
	return err
}
