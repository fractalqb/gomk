package gomk

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
)

type TaskEnv struct {
	Ctx      context.Context
	Trace    Tracer
	In       io.Reader
	Out, Err io.Writer
}

type Task interface {
	ErrStater

	Project() *Project
	Name() string

	WorkDir(path ...string) Task
	ChangeEnv(set map[string]string, unset ...string) Task
	DependOn(taskname ...string) Task

	Run(env TaskEnv) error
	DependsOn() []string
}

type TaskBase struct {
	Err      error
	name     string
	prj      *Project
	wdir     []string
	envSet   map[string]string
	envUnset []string
	depNames []string
}

func (t *TaskBase) ErrState() error { return t.Err }

func (t *TaskBase) Project() *Project { return t.prj }

func (t *TaskBase) Name() string { return t.name }

func (t *TaskBase) WorkDir(path ...string) { t.wdir = path }

func (t *TaskBase) DependOn(taskname ...string) {
NEXT_DEP:
	for _, dn := range taskname {
		for _, d := range t.depNames {
			if d == dn {
				continue NEXT_DEP
			}
		}
		t.depNames = append(t.depNames, dn)
	}
}

func (t *TaskBase) ChangeEnv(set map[string]string, unset ...string) {
	if t.envSet == nil {
		t.envSet = make(map[string]string)
	}
	for k, v := range set {
		t.envSet[k] = v
	}
NEXT_UNSET:
	for _, u := range unset {
		for _, eu := range t.envUnset {
			if u == eu {
				continue NEXT_UNSET
			}
		}
		t.envUnset = append(t.envUnset, u)
	}
}

func (t *TaskBase) DependsOn() []string { return t.depNames }

type NopTask struct{ TaskBase }

func NewNopTask(onErr OnErrFunc, p *Project, name string) *NopTask {
	if _, ok := p.tasks[name]; ok {
		t := &NopTask{TaskBase{Err: fmt.Errorf("redefining task '%s'", name)}}
		CheckErrState(onErr, t)
		return t
	}
	t := &NopTask{TaskBase: TaskBase{prj: p, name: name}}
	p.tasks[name] = t
	return t
}

func (*NopTask) Run(TaskEnv) error { return nil }

func (t *NopTask) WorkDir(path ...string) Task {
	t.TaskBase.WorkDir(path...)
	return t
}

func (t *NopTask) ChangeEnv(set map[string]string, unset ...string) Task {
	t.TaskBase.ChangeEnv(set, unset...)
	return t
}

func (t *NopTask) DependOn(taskname ...string) Task {
	t.TaskBase.DependOn(taskname...)
	return t
}

type CmdTask struct {
	TaskBase
	Command CommandDef
}

func NewCmdTask(onErr OnErrFunc, p *Project, name string, cmd string, arg ...string) *CmdTask {
	return NewCmdDefTask(onErr, p, name, CommandDef{cmd, arg})
}

func NewCmdDefTask(onErr OnErrFunc, p *Project, name string, def CommandDef) *CmdTask {
	if _, ok := p.tasks[name]; ok {
		t := &CmdTask{TaskBase: TaskBase{Err: fmt.Errorf("redefineing task '%s'", name)}}
		CheckErrState(onErr, t)
		return t
	}
	t := &CmdTask{
		TaskBase: TaskBase{prj: p, name: name},
		Command:  def,
	}
	p.tasks[name] = t
	return t
}

func (t *CmdTask) Run(tenv TaskEnv) error {
	ctx, cv, err := SubContext(
		t.Project().EnvContext(tenv.Ctx),
		filepath.Join(t.wdir...),
		t.envSet,
		t.envUnset...,
	)
	if err != nil {
		return err
	}
	cmd := DefinedCommand(ctx, t.Command)
	TraceStart(tenv.Trace, cv.Dir, cmd)
	err = cmd.Run()
	if err != nil {
		TraceFail(tenv.Trace, err, cv.Dir, cmd)
	} else {
		TraceDone(tenv.Trace, cv.Dir, cmd)
	}
	return err
}

func (t *CmdTask) WorkDir(path ...string) Task {
	t.TaskBase.WorkDir(path...)
	return t
}

func (t *CmdTask) ChangeEnv(set map[string]string, unset ...string) Task {
	t.TaskBase.ChangeEnv(set, unset...)
	return t
}

func (t *CmdTask) DependOn(taskname ...string) Task {
	t.TaskBase.DependOn(taskname...)
	return t
}

type PipeTask struct {
	TaskBase
	Pipe []CommandDef
}

func (t *PipeTask) Run(tenv TaskEnv) error {
	ctx, _, err := SubContext(
		t.Project().EnvContext(tenv.Ctx),
		filepath.Join(t.wdir...),
		t.envSet,
		t.envUnset...,
	)
	if err != nil {
		return err
	}
	p := BuildPipe(ctx)
	for _, cdef := range t.Pipe {
		p.Command(cdef.Name, cdef.Args...)
	}
	return p.Run()
}

func (t *PipeTask) WorkDir(path ...string) Task {
	t.TaskBase.WorkDir(path...)
	return t
}

func (t *PipeTask) ChangeEnv(set map[string]string, unset ...string) Task {
	t.TaskBase.ChangeEnv(set, unset...)
	return t
}

func (t *PipeTask) DependOn(taskname ...string) Task {
	t.TaskBase.DependOn(taskname...)
	return t
}

type TaskFunc func(context.Context, RunEnv) error

type CodeTask struct {
	TaskBase
	f TaskFunc
}

func NewCodeTask(onErr OnErrFunc, p *Project, name string, f TaskFunc) *CodeTask {
	if _, ok := p.tasks[name]; ok {
		t := &CodeTask{TaskBase: TaskBase{Err: fmt.Errorf("redefineing task '%s'", name)}}
		CheckErrState(onErr, t)
		return t
	}
	t := &CodeTask{
		TaskBase: TaskBase{prj: p, name: name},
		f:        f,
	}
	p.tasks[name] = t
	return t

}

func (t *CodeTask) Run(rt TaskEnv) error {
	ctx, cv, err := SubContext(
		t.Project().EnvContext(rt.Ctx),
		filepath.Join(t.wdir...),
		t.envSet,
		t.envUnset...,
	)
	if err != nil {
		return err
	}
	cv.SetIO(false, rt.In, rt.Out, rt.Err)
	return t.f(ctx, *cv)
}

func (t *CodeTask) WorkDir(path ...string) Task {
	t.TaskBase.WorkDir(path...)
	return t
}

func (t *CodeTask) ChangeEnv(set map[string]string, unset ...string) Task {
	t.TaskBase.ChangeEnv(set, unset...)
	return t
}

func (t *CodeTask) DependOn(taskname ...string) Task {
	t.TaskBase.DependOn(taskname...)
	return t
}
