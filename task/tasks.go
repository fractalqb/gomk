package task

import (
	"os/exec"

	"git.fractalqb.de/fractalqb/gomk"
)

func MkGetTool(build *gomk.Build, exe, repo string) error {
	_, err := exec.LookPath(exe)
	if err == nil {
		return err
	}
	build.WithEnv(func(e *gomk.Env) {
		e.Set("GO111MODULE", "on")
	}, func() {
		build.WDir().Exec("go", "get", "-u", repo)
	})
	return nil
}

func GetStringer(build *gomk.Build) {
	err := MkGetTool(build, "stringer", "golang.org/x/tools/cmd/stringer")
	if err != nil {
		panic(err)
	}
}

func GetVersioner(build *gomk.Build) {
	err := MkGetTool(build, "versioner", "git.fractalqb.de/fractalqb/pack/versioner")
	if err != nil {
		panic(err)
	}
}

func DepsGraph(build *gomk.Build) {
	err := MkGetTool(build, "modgraphviz", "golang.org/x/exp/cmd/modgraphviz")
	if err != nil {
		panic(err)
	}
	build.WithEnv(func(e *gomk.Env) {
		e.Set("GO111MODULE", "on")
	}, func() {
		build.WDir().ExecPipe(
			exec.Command("go", "mod", "graph"),
			exec.Command("modgraphviz"),
			exec.Command("dot", "-Tsvg", "-o", "depgraph.svg"),
		)
	})
}
