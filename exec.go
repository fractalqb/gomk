package gomk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type CmdOp struct {
	CWD             string
	Exe             string
	Args            []string
	InFile, OutFile string
	Desc            string
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
	if op.InFile != "" {
		if r, err := os.Open(op.InFile); err != nil {
			return err
		} else {
			defer r.Close()
			cmd.Stdin = r
		}
	} else {
		cmd.Stdin = env.In
	}
	if op.OutFile != "" {
		if w, err := os.Create(op.OutFile); err != nil {
			return err
		} else {
			defer w.Close()
			cmd.Stdout = w
		}
	} else {
		cmd.Stdout = env.Out
	}
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

type ConverCmd struct {
	Exe    string
	Output string // 1, 2, stdout, <output-flag> e.g. "-o"
	Args   []string
}

func (cc *ConverCmd) BuildAction(prj *Project, premises, results []*Goal) (*Action, error) {
	if len(premises) != 1 || len(results) != 1 {
		return nil, errors.New("ConvertCmd requires one premise and one result file goal")
	}
	pre, res := premises[0], results[0]
	inFile, ok := pre.Artefact.(File)
	if !ok {
		return nil, fmt.Errorf("ConvertCmd expect one premise file, have one %T", pre.Artefact)
	}
	outFile, ok := res.Artefact.(File)
	if !ok {
		return nil, fmt.Errorf("ConvertCmd expect one reslut file, have one %T", res.Artefact)
	}
	op := &CmdOp{
		CWD:  filepath.Dir(inFile.Path()),
		Exe:  cc.Exe,
		Args: cc.Args,
		Desc: fmt.Sprintf("%s: %s -> %s",
			filepath.Base(cc.Exe),
			filepath.Base(inFile.Path()),
			filepath.Base(outFile.Path()),
		),
	}
	if cc.Output != "" && cc.Output[0] == '-' {
		op.Args = append(op.Args, cc.Output, filepath.Base(outFile.Path()))
	} else if cc.Output == "1" {
		op.Args = append(op.Args, filepath.Base(outFile.Path()))
	}
	op.Args = append(op.Args, filepath.Base(inFile.Path()))
	if cc.Output == "2" {
		op.Args = append(op.Args, filepath.Base(outFile.Path()))
	} else if cc.Output == "stdout" {
		op.OutFile = outFile.Path()
	}
	return prj.NewAction(premises, results, op), nil
}
