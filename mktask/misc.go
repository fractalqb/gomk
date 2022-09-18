package mktask

import (
	"fmt"
	"os/exec"

	"git.fractalqb.de/fractalqb/gomk"
)

func getToolName(exe string) string {
	return fmt.Sprintf("get-tool:%s", exe)
}

func NewGetTool(onErr gomk.OnErrFunc, p *gomk.Project, update bool, exe, repo string) gomk.Task {
	name := getToolName(exe)
	if !update {
		if _, err := exec.LookPath(exe); err == nil {
			return gomk.NewNopTask(onErr, p, name)
		}
	}
	t := gomk.NewCmdTask(onErr, p, name, "go", "install", repo)
	if t.Err != nil {
		return t
	}
	t.ChangeEnv(map[string]string{"GO111MODULE": "on"})
	return t
}

func NewGetStringer(onErr gomk.OnErrFunc, p *gomk.Project, as string, update bool) gomk.Task {
	if as == "" {
		as = "get-stringer"
	}
	return NewGetTool(onErr, p, update, as,
		"golang.org/x/tools/cmd/stringer@latest",
	)
}

func NewGetVersioner(onErr gomk.OnErrFunc, p *gomk.Project, as string, update bool) gomk.Task {
	if as == "" {
		as = "get-versioner"
	}
	return NewGetTool(onErr, p, update, as,
		"git.fractalqb.de/fractalqb/pack/versioner@latest",
	)
}

func NewGetGoVulncheck(onErr gomk.OnErrFunc, p *gomk.Project, as string, update bool) gomk.Task {
	if as == "" {
		as = "get-vulncheck"
	}
	return NewGetTool(onErr, p, update,
		"vulncheck",
		"golang.org/x/vuln/cmd/govulncheck@latest",
	)
}
