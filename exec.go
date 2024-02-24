package gomk

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type CmdOp struct {
	CWD             string
	Exe             string
	Args            []string
	InFile, OutFile string
	Desc            string
}

var _ gomkore.Operation = (*CmdOp)(nil)

func (op *CmdOp) Describe(a *Action, _ *Env) string {
	if op.Desc == "" {
		path := filepath.Base(op.Exe)
		op.Desc = fmt.Sprintf("%s$%s%v", path, op.Exe, op.Args)
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

// TODO include environment values
func (op *CmdOp) WriteHash(h hash.Hash, a *Action, _ *Env) (bool, error) {
	fmt.Fprintln(h, op.CWD)
	fmt.Fprintln(h, op.Exe)
	for _, arg := range op.Args {
		fmt.Fprintln(h, arg)
	}
	fmt.Fprintln(h, op.InFile)
	fmt.Fprintln(h, op.OutFile)
	return true, nil
}

type PipeOp []CmdOp

var _ gomkore.Operation = PipeOp{}

func (po PipeOp) Describe(a *Action, env *Env) string {
	if len(po) == 0 {
		return "empty pipe"
	}
	var sb strings.Builder
	sb.WriteString(po[0].Describe(a, env))
	for _, o := range po[1:] {
		sb.WriteByte('|')
		sb.WriteString(o.Describe(a, env))
	}
	return sb.String()
}

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

func (po PipeOp) WriteHash(h hash.Hash, a *Action, env *Env) (bool, error) {
	for _, cmd := range po {
		ok, err := cmd.WriteHash(h, a, env)
		if !ok || err != nil {
			return false, err
		}
	}
	return true, nil
}

type ConvertCmd struct {
	Exe string
	// Output controls how the output of the convert command is to be written to
	// the result file.
	// "1"     : The output file name is put after .Args before the input file name.
	// "2"     : The output file name is put after input file name, i.e. as the last argument.
	// "stdout": Stdout is redirected to the output file.
	// "-"<opt>: Flags to set output file name, e.g. "-o".
	Output string
	Args   []string
}

var _ gomkore.Operation = (*ConvertCmd)(nil)

func (cc *ConvertCmd) Describe(*Action, *Env) string {
	return fmt.Sprintf("%s-Convert", filepath.Base(cc.Exe))
}

func (cc *ConvertCmd) Do(ctx context.Context, a *Action, env *Env) error {
	if len(a.Premises) != 1 || len(a.Results) != 1 {
		return errors.New("ConvertCmd requires one premise and one result file goal")
	}
	pre, res := a.Premises[0], a.Results[0]
	inFile, ok := pre.Artefact.(File)
	if !ok {
		return fmt.Errorf("ConvertCmd expect one premise file, have one %T", pre.Artefact)
	}
	outFile, ok := res.Artefact.(File)
	if !ok {
		return fmt.Errorf("ConvertCmd expect one reslut file, have one %T", res.Artefact)
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
	return op.Do(ctx, a, env)
}

func (cc *ConvertCmd) WriteHash(hash.Hash, *Action, *Env) (bool, error) {
	return false, errors.New("NYI: ConvertCmd.WriteHash()")
}
