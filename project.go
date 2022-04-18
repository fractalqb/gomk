package gomk

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type Config struct {
	RootDir string
	Env     interface{}
}

type Project struct {
	Env     EnvVars
	RootDir Dir
	Err     error
	tasks   map[string]Task
	parent  *Project
	subps   []*Project
}

func NewProject(onErr OnErrFunc, cfg *Config) (prj *Project) {
	if cfg == nil {
		var err error
		cfg = &Config{}
		cfg.RootDir, err = os.Getwd() // TODO filepath.Clean ?
		if err != nil {
			prj = &Project{Err: err}
			CheckErrState(onErr, prj)
			return prj
		}
	}
	prj = &Project{tasks: make(map[string]Task)}

	switch env := cfg.Env.(type) {
	case []string:
		e, err := NewEnvVarsString(env)
		if err != nil {
			prj.Err = err
			CheckErrState(onErr, prj)
			return prj
		}
		prj.Env = *e
	case map[string]string:
		prj.Env = *NewEnvVars(env)
	case nil:
		prj.Env = *NewOSEnv()
	default:
		prj.Err = fmt.Errorf("cannot set project env from %T", cfg.Env)
		CheckErrState(onErr, prj)
		return prj
	}

	cfg.RootDir, prj.Err = filepath.Abs(cfg.RootDir)
	if prj.Err != nil {
		CheckErrState(onErr, prj)
		return prj
	}
	prj.RootDir = Dir{prj: prj, abs: cfg.RootDir}

	return prj
}

func (prj *Project) ErrState() error { return prj.Err }

func (prj *Project) New(onErr OnErrFunc, cfg *Config) (sub *Project) {
	if cfg == nil {
		cfg = new(Config)
	}
	if cfg.Env == nil {
		strs := make([]string, len(prj.Env.Strings()))
		copy(strs, prj.Env.Strings())
		cfg.Env = strs
	}
	sub = NewProject(onErr, cfg)
	if sub.Err != nil {
		return sub
	}
	sub.parent = prj
	prj.subps = append(prj.subps, sub)
	return sub
}

func (prj *Project) Name() string {
	return filepath.Base(prj.RootDir.Abs())
}

func (prj *Project) Task(name string) Task { return prj.tasks[name] }

func (prj *Project) EnvContext(ctx context.Context) context.Context {
	res := context.WithValue(ctx, ctxValTag{}, &RunEnv{
		Dir: &prj.RootDir,
		Env: prj.Env,
	})
	return res
}

func (prj *Project) RootTasks(sorted bool) (ts []Task) {
	for _, t := range prj.tasks {
		if len(t.DependsOn()) == 0 {
			ts = append(ts, t)
		}
	}
	if sorted {
		sort.Slice(ts, func(i, j int) bool { return ts[i].Name() < ts[j].Name() })
	}
	return ts
}

func (prj *Project) LeafeTasks(sorted bool) (ts []Task) {
	rdeps := make(map[string]int)
	for _, t := range prj.tasks {
		if _, ok := rdeps[t.Name()]; !ok {
			rdeps[t.Name()] = 0
		}
		for _, dn := range t.DependsOn() {
			rdeps[dn] = rdeps[dn] + 1
		}
	}
	for tn, dn := range rdeps {
		if dn == 0 {
			ts = append(ts, prj.Task(tn))
		}
	}
	if sorted {
		sort.Slice(ts, func(i, j int) bool { return ts[i].Name() < ts[j].Name() })
	}
	return ts
}

func (prj *Project) WriteDeps(wr io.Writer) {
	var ts []Task
	for _, t := range prj.tasks {
		ts = append(ts, t)
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].Name() < ts[j].Name() })
	for _, t := range ts {
		fmt.Fprintf(wr, "%s:", t.Name())
		for _, dn := range t.DependsOn() {
			fmt.Fprintf(wr, " %s", dn)
		}
		fmt.Fprintln(wr)
	}
}

type Dir struct {
	prj  *Project
	pred *Dir
	abs  string
}

func (d *Dir) Abs() string { return d.abs }

func (d *Dir) Rel() (string, error) {
	if d.pred == nil {
		return ".", nil
	}
	return filepath.Rel(d.pred.Abs(), d.Abs())
}

func (d *Dir) Join(elem ...string) *Dir {
	joined := filepath.Join(append([]string{d.abs}, elem...)...)
	return &Dir{prj: d.prj, pred: d, abs: joined}
}

func (d *Dir) Back() *Dir { return d.pred }

func (d *Dir) Prj() *Project { return d.prj }
