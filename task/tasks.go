package task

import (
	"log"
	"os/exec"

	"git.fractalqb.de/fractalqb/gomk"
)

func MkGetTool(build *gomk.Build, update bool, exe, repo string) error {
	if !update {
		if _, err := exec.LookPath(exe); err == nil {
			return err
		}
	}
	build.WithEnv(func(e *gomk.Env) {
		e.Set("GO111MODULE", "on")
	}, func() {
		log.Printf("go get %s", repo)
		gomk.ExecOut(build.WDir(), nil, "go", "get", "-u", repo)
	})
	return nil
}

func GetStringer(build *gomk.Build, update bool) {
	err := MkGetTool(build, update, "stringer", "golang.org/x/tools/cmd/stringer")
	if err != nil {
		panic(err)
	}
}

func GetVersioner(build *gomk.Build, update bool) {
	err := MkGetTool(build, update, "versioner", "git.fractalqb.de/fractalqb/pack/versioner")
	if err != nil {
		panic(err)
	}
}

func DepsGraph(build *gomk.Build, update bool) {
	// const tool = "modgraphviz"
	// const tookpkg = "golang.org/x/exp/cmd/"+tool
	const tool = "gomodot"
	const tookpkg = "codeberg.org/fractalqb/" + tool
	build.WithEnv(func(e *gomk.Env) {
		e.Set("GO111MODULE", "on")
	}, func() {
		err := MkGetTool(build, update, tool, tookpkg)
		if err != nil {
			panic(err)
		}
		gomk.NewPipe(
			exec.Command("go", "mod", "graph"),
			exec.Command(tool),
			exec.Command("dot", "-Tsvg", "-o", "depgraph.svg"),
		).Exec(build.WDir())
	})
}
