package gomk

import (
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
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

type CmdOp struct {
	CWD             string
	Exe             string
	Args            []string
	InFile, OutFile string
	Desc            string
	UsesEnv         []string
}

var _ gomkore.Operation = (*CmdOp)(nil)

func (op *CmdOp) Describe(a *gomkore.Action, _ *gomkore.Env) string {
	if op.Desc == "" {
		op.Desc = fmt.Sprintf("%s$%s%v",
			op.CWD,
			filepath.Base(op.Exe),
			op.Args,
		)
	}
	return op.Desc
}

func (op *CmdOp) Do(tr *gomkore.Trace, a *gomkore.Action, env *gomkore.Env) error {
	xenv, err := env.ExecEnv()
	if err != nil {
		tr.Warn(err.Error(), slog.String("action", a.String()))
	}
	cmd := exec.CommandContext(tr.Ctx(), op.Exe, op.Args...)
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
	tr.Debug("exec `cmd` in `dir`",
		slog.String("cmd", cmd.String()),
		slog.String("dir", cmd.Dir),
	)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("command %s in %s failed: %w", cmd, cmd.Dir, err)
	}
	return err
}

// WriteHash considers env only if op.UsesEnv is set correctly.
func (op *CmdOp) WriteHash(h hash.Hash, a *gomkore.Action, env *gomkore.Env) (bool, error) {
	fmt.Fprintln(h, op.CWD)
	fmt.Fprintln(h, op.Exe)
	for _, arg := range op.Args {
		fmt.Fprintln(h, arg)
	}
	fmt.Fprintln(h, op.InFile)
	fmt.Fprintln(h, op.OutFile)
	for _, e := range op.UsesEnv {
		if v, ok := env.Tag(e); ok {
			fmt.Fprintf(h, "%s=%s\n", e, v)
		}
	}
	return true, nil
}

type PipeOp []CmdOp

var _ gomkore.Operation = PipeOp{}

func (po PipeOp) Describe(a *gomkore.Action, env *gomkore.Env) string {
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

func (po PipeOp) Do(tr *gomkore.Trace, a *gomkore.Action, env *gomkore.Env) error {
	var (
		cmds      = make([]*exec.Cmd, len(po))
		pipes     = make([]piperw, len(po)-1)
		xenv, err = env.ExecEnv()
	)
	if err != nil {
		tr.Warn(err.Error(), slog.String("action", a.String()))
	}
	for i := 0; i < len(po); i++ {
		cop := &po[i]
		cmd := exec.CommandContext(tr.Ctx(), cop.Exe, cop.Args...)
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
				if e := cmds[k].Process.Kill(); e != nil {
					tr.Warn("aborting pipe with `error`", `error`, e)
				}
			}
			return err
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			for k := i + 1; k < len(cmds); k++ {
				if e := cmds[k].Process.Kill(); e != nil {
					tr.Warn("aborting pipe with `error`", `error`, e)
				}
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

func (po PipeOp) WriteHash(h hash.Hash, a *gomkore.Action, env *gomkore.Env) (bool, error) {
	for _, cmd := range po {
		ok, err := cmd.WriteHash(h, a, env)
		if !ok || err != nil {
			return false, err
		}
	}
	return true, nil
}

type ConvertCmd struct {
	Exe  string
	Args []string

	// PassOut controls how the output of the convert command is to be passed to
	// the command:
	// "1"     : The output file name is put after .Args before the input file name.
	// "2"     : The output file name is put after input file name, i.e. as the last argument.
	// "stdout": Stdout is redirected to the output file.
	// "-"<opt>: Flags to set output file name, e.g. "-o".
	PassOut string

	// If the convert command interprets the output relative to the input
	// instead of the current working directory.
	OutRelToIn bool

	// If the convert command wants the output directiory instead of the output
	// file name.
	OutDir bool

	mkfs.MkDirs
}

var _ gomkore.Operation = (*ConvertCmd)(nil)

func (cc *ConvertCmd) Describe(*gomkore.Action, *gomkore.Env) string {
	return fmt.Sprintf("%s-Convert", filepath.Base(cc.Exe))
}

func (cc *ConvertCmd) Do(tr *gomkore.Trace, a *gomkore.Action, env *gomkore.Env) error {
	prj := a.Project()
	cwd, err := prj.AbsPath("")
	if err != nil {
		return err
	}

	var pre *gomkore.Goal
	if tpre, err := Goals(a.Premises(), true, Tangible, AType[mkfs.File]); err != nil {
		return fmt.Errorf("convert command premise: %w", err)
	} else if len(tpre) != 1 {
		return errors.New("convert command requires one file premise")
	} else {
		pre = tpre[0]
	}
	inFile, err := prj.RelPathTo(cwd, pre.Artefact.(mkfs.File).Path())
	if err != nil {
		return fmt.Errorf("convert command input: %w", err)
	}

	var (
		outFile string
		outPath = func(a mkfs.Artefact) (string, error) {
			if cc.OutRelToIn {
				return prj.RelPathTo(filepath.Dir(inFile), a.Path())
			}
			return prj.RelPathTo(cwd, a.Path())
		}
	)
	res, err := Goals(a.Results(), true, Tangible, AType[mkfs.Artefact])
	if err != nil {
		return fmt.Errorf("convert command result: %w", err)
	} else if len(res) == 0 {
		return errors.New("convert command without result")
	} else if cc.OutDir {
		if outFile, err = outPath(res[0].Artefact.(mkfs.Artefact)); err != nil {
			return err
		} else if _, ok := res[0].Artefact.(mkfs.File); ok {
			outFile = filepath.Dir(outFile)
		}
		for _, r := range res[1:] {
			of, err := outPath(r.Artefact.(mkfs.Artefact))
			if err != nil {
				return err
			}
			if _, ok := r.Artefact.(mkfs.Artefact); ok {
				of = filepath.Dir(of)
			}
			if of != outFile {
				return errors.New("convert command with inconsistent output dirs")
			}
		}
	} else {
		if len(res) != 1 {
			return errors.New("convert command requires one file result")
		}
		if f, ok := res[0].Artefact.(mkfs.File); !ok {
			return fmt.Errorf("convert command requires one file result, have %T", res[0].Artefact)
		} else if outFile, err = outPath(f); err != nil {
			return err
		}
	}

	op := &CmdOp{
		CWD:  cwd,
		Exe:  cc.Exe,
		Args: cc.Args,
		Desc: fmt.Sprintf("%s: %s -> %s",
			filepath.Base(cc.Exe),
			filepath.Base(inFile),
			filepath.Base(outFile),
		),
	}
	if cc.PassOut != "" && cc.PassOut[0] == '-' {
		op.Args = append(op.Args, cc.PassOut, outFile)
	} else if cc.PassOut == "1" {
		op.Args = append(op.Args, outFile)
	}
	op.Args = append(op.Args, inFile)
	switch cc.PassOut {
	case "2":
		op.Args = append(op.Args, outFile)
	case "stdout":
		op.OutFile = outFile
	}
	if err := cc.MkDirs.Do(tr, a, env); err != nil {
		return err
	}
	return op.Do(tr, a, env)
}

func (cc *ConvertCmd) WriteHash(hash.Hash, *gomkore.Action, *gomkore.Env) (bool, error) {
	return false, errors.New("NYI: ConvertCmd.WriteHash()")
}
