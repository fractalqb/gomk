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
	GoExe string
}

func (t *GoTool) goExe() (string, error) {
	if t.GoExe != "" {
		return t.GoExe, nil
	}
	return exec.LookPath("go")
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

var _ ActionBuilder = (*GoBuild)(nil)

func (gb *GoBuild) BuildAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
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
	case DirList:
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
		CWD:  prj.Dir,
		Exe:  goTool,
		Args: []string{"build", "-C", chdir},
		Desc: fmt.Sprintf("%s$ go build", chdir),
	}
	if gb.Install {
		op.Args[0] = "install"
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
	return prj.NewAction(ps, rs, op), nil
}

type GoTest struct {
	GoTool
	CWD  string
	Pkgs []string
}

var _ ActionBuilder = (*GoTest)(nil)

func (gt *GoTest) BuildAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
	goTool, err := gt.goExe()
	if err != nil {
		return nil, err
	}
	op := &CmdOp{
		CWD:  prj.Dir,
		Exe:  goTool,
		Args: []string{"test"},
		Desc: fmt.Sprintf("go test %s", strings.Join(gt.Pkgs, " ")),
	}
	if len(gt.Pkgs) > 0 {
		op.Args = append(op.Args, gt.Pkgs...)
	}
	return prj.NewAction(ps, rs, op), nil
}

type GoGenerate struct {
	GoTool
	CWD       string
	FilesPkgs []string
	Run       string
	Skip      string
}

var _ ActionBuilder = (*GoGenerate)(nil)

func (gg *GoGenerate) BuildAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
	goTool, err := gg.goExe()
	if err != nil {
		return nil, err
	}
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
	return prj.NewAction(ps, rs, op), nil
}

type GoRun struct {
	GoTool
	CWD  string
	Exec string
	Pkg  string
	Args []string
}

var _ ActionBuilder = (*GoRun)(nil)

func (gr *GoRun) BuildAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	var err error
	if gr.Pkg == "" {
		return nil, errors.New("go run without package")
	}
	goTool, err := gr.goExe()
	if err != nil {
		return nil, err
	}
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
	return prj.NewAction(ps, rs, op), nil
}

type GoVulncheck struct {
	Version  string
	CWD      string
	Tags     []string
	Patterns []string
}

var _ ActionBuilder = (*GoVulncheck)(nil)

func (gvc *GoVulncheck) BuildAction(prj *Project, ps, rs []*Goal) (*Action, error) {
	const govulncheck = "golang.org/x/vuln/cmd/govulncheck"
	var err error
	goTool, err := (&GoTool{}).goExe()
	if err != nil {
		return nil, err
	}
	op := &CmdOp{
		CWD:  prj.Dir,
		Exe:  goTool,
		Args: []string{"run"},
		Desc: "govulncheck",
	}
	if len(gvc.Patterns) > 0 {
		op.Desc = fmt.Sprintf("%s %s", op.Desc, strings.Join(gvc.Patterns, " "))
	}
	if len(gvc.Tags) > 0 {
		op.Args = append(op.Args, "-tags", strings.Join(gvc.Tags, ","))
	}
	if gvc.Version == "" {
		op.Args = append(op.Args, govulncheck+"@latest")
	} else {
		op.Args = append(op.Args, govulncheck+"@"+gvc.Version)
	}
	op.Args = append(op.Args, gvc.Patterns...)
	return prj.NewAction(ps, rs, op), nil
}
