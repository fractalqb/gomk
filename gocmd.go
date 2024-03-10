package gomk

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"os/exec"
	"path/filepath"
	"strings"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

type GoTool struct {
	GoExe string
}

func (t *GoTool) goExe() (string, error) {
	if t.GoExe != "" {
		return t.GoExe, nil
	}
	return exec.LookPath("go")
}

func (t *GoTool) describe(base string, a *Action) string {
	if a == nil || len(a.Results()) == 0 {
		return "Go " + base
	}
	s := fmt.Sprintf("Go %s %s", base, a.Result(0).Name())
	if len(a.Results()) > 1 {
		s += "…"
	}
	return s
}

// GoBuild is an [ActionBuilder] that expects exactly one result [Goal] with
// either a [File] or [Directory] artefact. The created action will run 'go
// build' with -C to change into the result's deirectory.
type GoBuild struct {
	GoTool
	Install  bool
	TrimPath bool
	LDFlags  []string // See https://pkg.go.dev/cmd/link
	SetVars  []string // See https://pkg.go.dev/cmd/link Flag: -X
}

var _ gomkore.Operation = (*GoBuild)(nil)

func (gb *GoBuild) Describe(a *Action, _ *Env) string {
	return gb.describe("build", a)
}

func (gb *GoBuild) Do(ctx context.Context, a *Action, env *Env) error {
	res, _ := Goals(a.Results(), false, Tangible)
	if len(res) > 1 {
		var sb strings.Builder
		fmt.Fprintf(&sb, "go build with %d results:", len(a.Results()))
		for _, r := range a.Results() {
			fmt.Fprintf(&sb, " %s", r)
		}
		return errors.New(sb.String())
	}
	pkgs, err := Goals(a.Premises(), true, Tangible, AType[mkfs.Directory])
	if err != nil {
		return fmt.Errorf("go build premises: %w", err)
	}
	prj := a.Project()
	goTool, err := gb.goExe()
	if err != nil {
		return err
	}
	op := &CmdOp{CWD: prj.Dir, Exe: goTool, Args: []string{"build"}}
	if gb.Install {
		op.Args[0] = "install"
	}
	if len(res) == 1 {
		fs, ok := res[0].Artefact.(mkfs.Artefact)
		if !ok {
			return fmt.Errorf("invalid go buidl result type %T", res[0])
		}
		op.Args = append(op.Args, "-o", fs.Path())
	}
	if gb.TrimPath {
		op.Args = append(op.Args, "-trimpath")
	}
	var ldFlags strings.Builder
	for _, f := range gb.LDFlags {
		if f != "" {
			if ldFlags.Len() > 0 {
				ldFlags.WriteByte(' ')
			}
			ldFlags.WriteString(f)
		}
	}
	for _, v := range gb.SetVars {
		if ldFlags.Len() > 0 {
			ldFlags.WriteByte(' ')
		}
		fmt.Fprintf(&ldFlags, " -X %s", v)
	}
	if ldFlags.Len() > 0 {
		op.Args = append(op.Args, "-ldflags", ldFlags.String())
	}
	for _, pkg := range pkgs {
		dir := pkg.Artefact.(mkfs.Directory).Path()
		if dir, err = prj.RelPath(dir); err != nil {
			return fmt.Errorf("go build package: %w", err)
		}
		op.Args = append(op.Args, "./"+dir)
	}
	return op.Do(ctx, a, env)
}

func (*GoBuild) WriteHash(hash.Hash, *Action, *Env) (bool, error) {
	return false, errors.New("NYI: GoBuild.WriteHash()")
}

type GoTest struct {
	GoTool
	CWD  string
	Pkgs []string // TODO use action prems instead
}

var _ gomkore.Operation = (*GoTest)(nil)

func (gt *GoTest) Describe(a *Action, _ *Env) string {
	return gt.describe("test", a)
}

func (gt *GoTest) Do(ctx context.Context, a *Action, env *Env) error {
	var err error
	goTool, err := gt.goExe()
	if err != nil {
		return err
	}
	prj := a.Project()
	op := &CmdOp{
		CWD:  prj.Dir,
		Exe:  goTool,
		Args: []string{"test"},
		Desc: fmt.Sprintf("go test %s", strings.Join(gt.Pkgs, " ")),
	}
	if len(gt.Pkgs) > 0 {
		op.Args = append(op.Args, gt.Pkgs...)
	}
	return op.Do(ctx, a, env)
}

func (*GoTest) WriteHash(hash.Hash, *Action, *Env) (bool, error) {
	return false, errors.New("NYI: GoTest.WriteHash()")
}

type GoGenerate struct {
	GoTool
	CWD       string
	FilesPkgs []string // TODO use action prems instead
	Run       string
	Skip      string
}

var _ gomkore.Operation = (*GoGenerate)(nil)

func (gg *GoGenerate) Describe(a *Action, _ *Env) string {
	return gg.describe("generate", a)
}

func (gg *GoGenerate) Do(ctx context.Context, a *Action, env *Env) error {
	var err error
	goTool, err := gg.goExe()
	if err != nil {
		return err
	}
	prj := a.Project()
	op := &CmdOp{
		CWD:  prj.Dir,
		Exe:  goTool,
		Args: []string{"generate"},
		Desc: fmt.Sprintf("go generate %s", strings.Join(gg.FilesPkgs, " ")),
	}
	if gg.CWD != "" {
		op.CWD = filepath.Join(op.CWD, gg.CWD)
	}
	if gg.Run != "" {
		op.Args = append(op.Args, "-run", gg.Run)
	}
	if gg.Skip != "" {
		op.Args = append(op.Args, gg.Skip)
	}
	if len(gg.FilesPkgs) > 0 {
		op.Args = append(op.Args, gg.FilesPkgs...)
	}
	return op.Do(ctx, a, env)
}

func (*GoGenerate) WriteHash(hash.Hash, *Action, *Env) (bool, error) {
	return false, errors.New("NYI: GoGenerate.WriteHash()")
}

type GoRun struct {
	GoTool
	CWD  string
	Exec string
	Pkg  string // TODO use action prems instead (>1)
	Args []string
}

var _ gomkore.Operation = (*GoRun)(nil)

func (gr *GoRun) Describe(a *Action, _ *Env) string {
	return gr.describe("run", a)
}

func (gr *GoRun) Do(ctx context.Context, a *Action, env *Env) error {
	var err error
	if gr.Pkg == "" {
		return errors.New("go run without package")
	}
	goTool, err := gr.goExe()
	if err != nil {
		return err
	}
	prj := a.Project()
	op := &CmdOp{
		CWD:  prj.Dir,
		Exe:  goTool,
		Args: []string{"run"},
		Desc: fmt.Sprintf("go run %s", gr.Pkg),
	}
	if gr.CWD != "" {
		op.CWD = filepath.Join(op.CWD, gr.CWD)
	}
	if gr.Exec != "" {
		op.Args = append(op.Args, "-exec", gr.Exec)
	}
	op.Args = append(op.Args, gr.Pkg)
	op.Args = append(op.Args, gr.Args...)
	return op.Do(ctx, a, env)
}

func (*GoRun) WriteHash(hash.Hash, *Action, *Env) (bool, error) {
	return false, errors.New("NYI: GoRun.WriteHash()")
}
