package gomk

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
)

type CmdOp struct {
	CWD  string
	Exe  string
	Args []string
	Desc string
}

func (op *CmdOp) Describe(prj *Project) string {
	if op.Desc == "" {
		path := prj.relPath(op.CWD)
		return fmt.Sprintf("%s$%s%v", path, op.Exe, op.Args)
	}
	return op.Desc
}

func (op *CmdOp) Do(ctx context.Context, a *Action, env *Env) error {
	xenv, err := env.ExecEnv()
	if err != nil {
		env.Log.Warn(err.Error(), slog.String("action", a.String()))
	}
	cmd := exec.CommandContext(ctx, op.Exe, op.Args...)
	cmd.Dir = op.CWD
	cmd.Env = xenv
	cmd.Stdin = env.In
	cmd.Stdout = env.Out
	cmd.Stderr = env.Err
	env.Log.Debug("exec `cmd` in `dir`",
		slog.String("cmd", cmd.String()),
		slog.String("dir", cmd.Dir),
	)
	err = cmd.Run()
	if err != nil {
		env.Log.Error("failed `cmd` in `dir` with `error`",
			slog.String("cmd", cmd.String()),
			slog.String("dir", cmd.Dir),
			slog.String("error", err.Error()),
		)
	} else {
		env.Log.Debug("done with `cmd`", slog.String("cmd", cmd.String()))
	}
	return err
}

type PipeOp []CmdOp

func (po PipeOp) Do(ctx context.Context, a *Action, env *Env) error {
	var (
		cmds      = make([]*exec.Cmd, len(po))
		pipes     = make([]piperw, len(po)-1)
		xenv, err = env.ExecEnv()
	)
	if err != nil {
		env.Log.Warn(err.Error(), slog.String("action", a.String()))
	}
	for i := 0; i < len(po); i++ {
		cop := &po[i]
		cmd := exec.CommandContext(ctx, cop.Exe, cop.Args...)
		cmd.Dir = cop.CWD
		cmd.Env = xenv
		if i == 0 {
			cmd.Stdin = env.In
		} else {
			r, w := io.Pipe()
			cmds[i-1].Stdout = w
			cmd.Stdin = r
			pipes[i-1] = piperw{r, w}
		}
		if i+1 == len(po) {
			cmd.Stdout = env.Out
		}
		cmd.Stderr = env.Err
		cmds[i] = cmd
	}
	for i, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			for k := 0; k < i; k++ {
				cmds[k].Process.Kill() // TODO check
			}
			return err
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			for k := i + 1; k < len(cmds); k++ {
				cmds[k].Process.Kill() // TODO check
			}
			return err
		}
		if i < len(pipes) {
			pipes[i].w.Close()
		}
	}
	return nil
}

type piperw struct {
	r *io.PipeReader
	w *io.PipeWriter
}
