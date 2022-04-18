package gomk

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

func ValidEnvKey(k string) error {
	if strings.IndexByte(k, '=') >= 0 {
		return errors.New("env key contains '='")
	}
	return nil
}

type EnvVars struct {
	kvm map[string]string
	env []string
}

func NewEnvVars(init map[string]string) *EnvVars {
	res := &EnvVars{kvm: init}
	if res.kvm == nil {
		res.kvm = make(map[string]string)
	}
	return res
}

func NewEnvVarsString(init []string) (*EnvVars, error) {
	env := NewEnvVars(nil)
	env.env = init
	for _, str := range init {
		sep := strings.IndexByte(str, '=')
		k, v := str[:sep], str[sep+1:]
		if err := ValidEnvKey(k); err != nil {
			return nil, err
		}
		env.kvm[k] = v
	}
	return env, nil
}

func NewOSEnv() *EnvVars {
	res, _ := NewEnvVarsString(os.Environ())
	return res
}

func (e EnvVars) Get(key string) string { return e.kvm[key] }

func (e EnvVars) Lookup(key string) (val string, ok bool) {
	val, ok = e.kvm[key]
	return val, ok
}

func (e *EnvVars) Set(key, val string) *EnvVars {
	e.kvm[key] = val
	e.env = nil
	return e
}

func (e *EnvVars) Unset(key string) *EnvVars {
	delete(e.kvm, key)
	e.env = nil
	return e
}

func (e *EnvVars) Strings() []string {
	if len(e.kvm) != len(e.env) {
		e.env = make([]string, 0, len(e.kvm))
		for k, v := range e.kvm {
			var sb strings.Builder
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(v)
			e.env = append(e.env, sb.String())
		}
	}
	return e.env
}

type Cmd = exec.Cmd

func Command(ctx context.Context, name string, arg ...string) *Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	if cv := CtxEnv(ctx); cv != nil {
		cmd.Dir = cv.Dir.Abs()
		cmd.Env = cv.Env.Strings()
		if cv.In == nil {
			cmd.Stdin = os.Stdin
		} else {
			cmd.Stdin = cv.In
		}
		if cv.Out == nil {
			cmd.Stdout = os.Stdout
		} else {
			cmd.Stdout = cv.Out
		}
		if cv.Err == nil {
			cmd.Stderr = os.Stderr
		} else {
			cmd.Stderr = cv.Err
		}
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}

type CmdDef struct {
	Name string
	Args []string
}

func CommandDef(ctx context.Context, cmd CmdDef) *Cmd {
	return Command(ctx, cmd.Name, cmd.Args...)
}

type Pipe struct {
	ctx   context.Context
	cmds  []*exec.Cmd
	stdin io.Reader
	pipes []piperw
}

type piperw struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func BuildPipeContext(ctx context.Context, stdin io.Reader) *Pipe {
	return &Pipe{ctx: ctx, stdin: stdin}
}

func (p *Pipe) Command(stderr io.Writer, name string, arg ...string) *Pipe {
	cmd := Command(p.ctx, name, arg...)
	p.cmds = append(p.cmds, cmd)
	if len(p.cmds) == 1 {
		cmd.Stdin = p.stdin
	}
	cmd.Stderr = stderr
	return p
}

func (p *Pipe) SetStdout(w io.Writer) *Pipe {
	if len(p.cmds) > 0 {
		p.cmds[len(p.cmds)-1].Stdout = w
	}
	return p
}

func (p Pipe) Run() error {
	if err := p.Start(); err != nil {
		return err
	}
	return p.Wait()
}

func (p *Pipe) Start() error {
	switch len(p.cmds) {
	case 0:
		return nil
	case 1:
		return p.cmds[0].Start()
	}
	pred := p.cmds[0]
	for _, cmd := range p.cmds[1:] {
		r, w := io.Pipe()
		p.pipes = append(p.pipes, piperw{r, w})
		pred.Stdout, cmd.Stdin = w, r
		pred = cmd
	}
	for _, cmd := range p.cmds {
		if err := cmd.Start(); err != nil {
			return err // TODO Howto clean up???
		}
	}
	return nil
}

func (p *Pipe) Wait() (err error) {
	switch len(p.cmds) {
	case 0:
		return nil
	case 1:
		err := p.cmds[0].Wait()
		return err
	}
	for i, cmd := range p.cmds {
		if e := cmd.Wait(); err == nil {
			err = e
		}
		if i != len(p.pipes) {
			prw := &p.pipes[i]
			prw.w.Close()
		}
	}
	for _, prw := range p.pipes {
		prw.r.Close()
	}
	return err
}
