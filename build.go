package gomk

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Env struct {
	parent *Env
	unset  map[string]bool
	local  map[string]string
	cache  []string
}

func NewEnvOS(osenv []string) (*Env, error) {
	res := new(Env)
	for _, oe := range osenv {
		sep := strings.IndexByte(oe, '=')
		if sep < 0 {
			return res, fmt.Errorf("invalid os env entry: '%s'", oe)
		}
		res.Set(oe[:sep], oe[sep+1:])
	}
	return res, nil
}

func (env *Env) Set(key, val string) {
	if env.unset != nil {
		delete(env.unset, key)
	}
	if env.local == nil {
		env.local = make(map[string]string)
	}
	env.local[key] = val
	env.cache = nil
}

func (env *Env) SetMap(e map[string]string) {
	for k, v := range e {
		env.Set(k, v)
	}
}

func (env *Env) Unset(keys ...string) {
	for _, key := range keys {
		if env.local != nil {
			delete(env.local, key)
		}
		if env.unset == nil {
			env.unset = make(map[string]bool)
		}
		env.unset[key] = true
	}
	env.cache = nil
}

func (env *Env) Get(key string) (string, bool) {
	for env != nil {
		if env.unset != nil {
			if _, ok := env.unset[key]; ok {
				return "", false
			}
		}
		if env.local != nil {
			if v, ok := env.local[key]; ok {
				return v, true
			}
		}
		env = env.parent
	}
	return "", false
}

func (env *Env) CmdEnv() []string {
	if env.cache == nil {
		if env.parent == nil && env.unset == nil && env.local == nil {
			return nil
		}
		merged := make(map[string]string)
		env.merge(merged)
		env.cache = make([]string, 0, len(merged))
		var sb strings.Builder
		for k, v := range merged {
			sb.Reset()
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(v)
			env.cache = append(env.cache, sb.String())
		}
	}
	return env.cache
}

func (env *Env) merge(m map[string]string) {
	if env.parent != nil {
		env.parent.merge(m)
	}
	if env.unset != nil {
		for k, _ := range env.unset {
			delete(m, k)
		}
	}
	if env.local != nil {
		for k, v := range env.local {
			m[k] = v
		}
	}
}

func (env *Env) New() *Env { return &Env{parent: env} }

func (env *Env) Push() {
	tmp := new(Env)
	*tmp = *env
	env.parent = tmp
	env.unset = nil
	env.local = nil
	env.cache = nil
}

func (env *Env) Pop() bool {
	if env.parent == nil {
		return false
	}
	*env = *env.parent
	return true
}

type Build struct {
	PrjRoot string
	Env     Env
}

func NewBuild(rootDir string, osEnv []string) (res *Build, err error) {
	if rootDir == "" {
		if rootDir, err = os.Getwd(); err != nil {
			return nil, err
		}
	} else if !filepath.IsAbs(rootDir) {
		if rootDir, err = filepath.Abs(rootDir); err != nil {
			return nil, err
		}
	}
	res = &Build{PrjRoot: rootDir}
	if osEnv != nil {
		oe, err := NewEnvOS(osEnv)
		if err != nil {
			return nil, err
		}
		res.Env = *oe
	}
	return res, nil
}

func MustNewBuild(rootDir string, osEnv []string) *Build {
	res, err := NewBuild(rootDir, osEnv)
	if err != nil {
		panic(err)
	}
	return res
}

func (b *Build) WDir() *WDir {
	res := &WDir{b: b, dir: b.PrjRoot}
	res.pre = res
	return res
}

func (b *Build) Rel(path string) (string, error) {
	return filepath.Rel(b.PrjRoot, path)
}

func (b *Build) WithEnv(change func(*Env), do func()) {
	b.Env.Push()
	defer b.Env.Pop()
	change(&b.Env)
	do()
}

func Try(f func()) (err error) {
	defer func() {
		if p := recover(); p != nil {
			switch e := p.(type) {
			case error:
				err = e
			case string:
				err = errors.New(e)
			default:
				err = fmt.Errorf("%+v", e)
			}
		}
	}()
	f()
	return nil
}

func Step(d *WDir, what string, f func(dir *WDir)) {
	log.Printf("do in %s: %s\n", d.MustRel(""), what)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d.MustRel(""), what, p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d.MustRel(""), what)
		}
	}()
	f(d)
}

func Exec(d *WDir, exe string, args ...string) {
	ExecOut(d, nil, exe, args...)
}

func ExecOut(d *WDir, out io.Writer, exe string, args ...string) {
	cmd := exec.Command(exe, args...)
	log.Printf("exec in %s: %s\n", d.MustRel(""), cmd)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d.MustRel(""), cmd, p)
			panic(p)
		} else {
			log.Printf("done in %s with: %s", d.MustRel(""), cmd)
		}
	}()
	cmd.Dir = d.dir
	cmd.Stdin = os.Stdin
	if out == nil {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = out
	}
	cmd.Stderr = os.Stderr
	cmd.Env = d.b.Env.CmdEnv()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func ExecFile(d *WDir, file string, exe string, args ...string) {
	wr, err := os.Create(d.Join(file))
	if err != nil {
		panic(err)
	}
	defer wr.Close()
	log.Printf("capture exec output to %s", file)
	ExecOut(d, wr, exe, args...)
}

type PipeError struct {
	act string
	cmd *exec.Cmd
	err error
}

func (pe PipeError) Unwrap() error { return pe.err }

func (pe PipeError) Error() string {
	return fmt.Sprintf("%s [%s]: %s", pe.act, pe.cmd, pe.err)
}

type Pipe struct {
	Cmds []*exec.Cmd
	Out  io.Writer
}

func NewPipe(cmds ...*exec.Cmd) *Pipe {
	return &Pipe{Cmds: cmds}
}

func (p Pipe) ExceFile(d *WDir, file string) {
	wr, err := os.Create(d.Join(file))
	if err != nil {
		panic(err)
	}
	defer wr.Close()
	log.Printf("capture next output to %s", file)
	p.Out = wr
	p.Exec(d)
}

func (p Pipe) Exec(d *WDir) {
	var sb strings.Builder
	l := len(p.Cmds)
	if l == 0 {
		return
	}
	pipew := make([]*io.PipeWriter, 0, l-1)
	for i, cmd := range p.Cmds {
		cmd.Dir = d.dir
		cmd.Stderr = os.Stderr
		cmd.Env = d.b.Env.CmdEnv()
		if i > 0 {
			pr, pw := io.Pipe()
			pipew = append(pipew, pw)
			p.Cmds[i-1].Stdout = pw
			cmd.Stdin = pr
			sb.WriteString(" | ")
		}
		fmt.Fprint(&sb, cmd.String())
	}
	log.Printf("pipe in %s: %s\n", d.MustRel(""), sb.String())
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d.MustRel(""), sb.String(), p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d.MustRel(""), sb.String())
		}
	}()
	p.Cmds[0].Stdin = os.Stdin
	if p.Out == nil {
		p.Cmds[l-1].Stdout = os.Stdout
	} else {
		p.Cmds[l-1].Stdout = p.Out
	}
	for _, cmd := range p.Cmds {
		if err := cmd.Start(); err != nil {
			panic(PipeError{act: "start", cmd: cmd, err: err})
		}
	}
	for i, cmd := range p.Cmds {
		if err := cmd.Wait(); err != nil {
			// TODO what about the rest .Wait() ?
			panic(PipeError{act: "wait", cmd: cmd, err: err})
		}
		if i < len(pipew) {
			pipew[i].Close()
		}
	}

}

func Update(d *WDir, target string, do func(newer []string) (int, error), deps ...string) (int, error) {
	var ttime time.Time
	if stat, err := os.Stat(d.Join(target)); os.IsNotExist(err) {
		n, err := do(deps)
		if n < 0 && err == nil {
			n = len(deps)
		}
		return n, err
	} else if err != nil {
		return -1, err
	} else {
		ttime = stat.ModTime()
	}
	var newer []string
	for _, dep := range deps {
		if stat, err := os.Stat(d.Join(dep)); err != nil {
			return -1, err
		} else if ttime.Before(stat.ModTime()) {
			newer = append(newer, dep)
		}
	}
	n, err := do(newer)
	if n < 0 && err == nil {
		n = len(newer)
	}
	return n, err
}
