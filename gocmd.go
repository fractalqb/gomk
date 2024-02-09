package gomk

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GoTool struct {
	Exe string
}

func (t *GoTool) goExe() (string, error) {
	if t.Exe != "" {
		return t.Exe, nil
	}
	return exec.LookPath("go")
}

type GoBuild struct {
	GoTool
	Install  bool
	TrimPath bool
	LDFlags  string
	SetVars  []string // See https://pkg.go.dev/cmd/link Flag: -X
}

func (gb *GoBuild) NewAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
	if prj, err = CheckProject(prj, ps, rs); err != nil {
		return nil, err
	}
	if len(rs) != 1 {
		var sb strings.Builder
		fmt.Fprintf(&sb, "go build with %d results:", len(rs))
		for _, r := range rs {
			fmt.Fprintf(&sb, " %s", r)
		}
		return nil, errors.New(sb.String())
	}
	var chdir string
	switch rs := rs[0].Artefact.(type) {
	case Directory:
		chdir = rs.Path()
	case File:
		chdir = filepath.Dir(rs.Path())
	default:
		return nil, fmt.Errorf("illegal type %T of go build result", rs)
	}
	if st, err := os.Stat(filepath.Join(prj.Dir, chdir)); err != nil {
		return nil, fmt.Errorf("path '%s' error: %w", chdir, err)
	} else if !st.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", chdir)
	}
	goTool, err := gb.goExe()
	if err != nil {
		return nil, err
	}
	op := &CmdOp{
		WorkDir: prj.Dir,
		Exe:     goTool,
		Args:    []string{"build", "-C", chdir},
		Desc:    fmt.Sprintf("go build %s", chdir),
	}
	if gb.Install {
		op.Args[0] = "install"
	}
	if gb.TrimPath {
		op.Args = append(op.Args, "-trimpath")
	}
	var ldflags strings.Builder
	if gb.LDFlags != "" {
		ldflags.WriteString(gb.LDFlags)
	}
	for _, v := range gb.SetVars {
		fmt.Fprintf(&ldflags, " -X %s", v)
	}
	op.Args = append(op.Args, "-ldflags", ldflags.String())
	return newAction(ps, rs, op), nil
}

type GoTest struct {
	GoTool
	CWD  string
	Pkgs []string
}

func (gt *GoTest) NewAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
	if prj, err = CheckProject(prj, ps, rs); err != nil {
		return nil, err
	}
	goTool, err := gt.goExe()
	if err != nil {
		return nil, err
	}
	op := &CmdOp{
		WorkDir: prj.Dir,
		Exe:     goTool,
		Args:    []string{"test"},
		Desc:    fmt.Sprintf("go test %s", strings.Join(gt.Pkgs, " ")),
	}
	if len(gt.Pkgs) > 0 {
		op.Args = append(op.Args, gt.Pkgs...)
	}
	return newAction(ps, rs, op), nil
}

type GoGenerate struct {
	GoTool
	CWD       string
	FilesPkgs []string
	Run       string
	Skip      string
}

func (gg *GoGenerate) NewAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
	if prj, err = CheckProject(prj, ps, rs); err != nil {
		return nil, err
	}
	goTool, err := gg.goExe()
	if err != nil {
		return nil, err
	}
	op := &CmdOp{
		WorkDir: prj.Dir,
		Exe:     goTool,
		Args:    []string{"generate"},
		Desc:    fmt.Sprintf("go generate %s", strings.Join(gg.FilesPkgs, " ")),
	}
	if gg.CWD != "" {
		op.WorkDir = filepath.Join(op.WorkDir, gg.CWD)
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
	return newAction(ps, rs, op), nil
}
