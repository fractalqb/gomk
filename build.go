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

func NewBuild(rootDir string, osEnv []string) (*Build, error) {
	res := &Build{
		PrjRoot: rootDir,
	}
	if res.PrjRoot == "" {
		var err error
		if res.PrjRoot, err = os.Getwd(); err != nil {
			return nil, err
		}
	}
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

func (b *Build) WithEnv(change func(*Env), do func()) {
	b.Env.Push()
	defer b.Env.Pop()
	change(&b.Env)
	do()
}

type WDir struct {
	b   *Build
	pre *WDir
	dir string
}

func (d *WDir) Build() *Build { return d.b }

func (d *WDir) Rel() string {
	res := d.dir[len(d.b.PrjRoot):]
	if res == "" {
		return "."
	}
	return res
}

func (d *WDir) Cd(dirs ...string) *WDir {
	tmp := make([]string, len(dirs)+1)
	tmp[0] = d.dir
	copy(tmp[1:], dirs)
	return &WDir{
		b:   d.b,
		pre: d,
		dir: filepath.Join(tmp...),
	}
}

func (d *WDir) Back() *WDir { return d.pre }

func (d *WDir) Do(what string, f func(dir *WDir)) {
	log.Printf("do in %s: %s\n", d.Rel(), what)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d.Rel(), what, p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d.Rel(), what)
		}
	}()
	f(d)
}

func (d *WDir) Exec(exe string, args ...string) {
	cmd := exec.Command(exe, args...)
	log.Printf("exec in %s: %s\n", d.Rel(), cmd)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d.Rel(), cmd, p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d.Rel(), cmd)
		}
	}()
	cmd.Dir = d.dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = d.b.Env.CmdEnv()
	if err := cmd.Run(); err != nil {
		panic(err)
	}
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

func (d *WDir) ExecPipe(cmds ...*exec.Cmd) {
	var sb strings.Builder
	l := len(cmds)
	if l == 0 {
		return
	}
	pipew := make([]*io.PipeWriter, 0, l-1)
	for i, cmd := range cmds {
		cmd.Dir = d.dir
		cmd.Stderr = os.Stderr
		cmd.Env = d.b.Env.CmdEnv()
		if i > 0 {
			pr, pw := io.Pipe()
			pipew = append(pipew, pw)
			cmds[i-1].Stdout = pw
			cmd.Stdin = pr
			sb.WriteString(" | ")
		}
		fmt.Fprint(&sb, cmd.String())
	}
	log.Printf("pipe in %s: %s\n", d.Rel(), sb.String())
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d.Rel(), sb.String(), p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d.Rel(), sb.String())
		}
	}()
	cmds[0].Stdin = os.Stdin
	cmds[l-1].Stdout = os.Stdout
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			panic(PipeError{act: "start", cmd: cmd, err: err})
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			// TODO what about the rest .Wait() ?
			panic(PipeError{act: "wait", cmd: cmd, err: err})
		}
		if i < len(pipew) {
			pipew[i].Close()
		}
	}
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
