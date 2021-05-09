package task

import (
	"log"
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
		log.Printf("go get %s", repo)
		gomk.ExecOut(build.WDir(), nil, "go", "get", "-u", repo)
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
	// const tool = "modgraphviz"
	// const tookpkg = "golang.org/x/exp/cmd/"+tool
	const tool = "gomodot"
	const tookpkg = "codeberg.org/fractalqb/" + tool
	err := MkGetTool(build, tool, tookpkg)
	if err != nil {
		panic(err)
	}
	build.WithEnv(func(e *gomk.Env) {
		e.Set("GO111MODULE", "on")
	}, func() {
		gomk.NewPipe(
			exec.Command("go", "mod", "graph"),
			exec.Command(tool),
			exec.Command("dot", "-Tsvg", "-o", "depgraph.svg"),
		).Exec(build.WDir())
	})
}
