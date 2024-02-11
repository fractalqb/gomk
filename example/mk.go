package main

import (
	"flag"
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/gomk"
)

var (
	// go generate ./...
	goGenerate = gomk.GoGenerate{FilesPkgs: []string{"./..."}}

	// go test ./...
	goTest = gomk.GoTest{Pkgs: []string{"./..."}}

	// govulncheck ./...
	goVulnchk = gomk.GoVulncheck{Patterns: []string{"./..."}}

	// go build -C <result-dir> -trimpath -s -w
	goBuild = gomk.GoBuild{TrimPath: true}

	writeDot bool
)

func flags() {
	fLog := flag.String("log", "", "Set log level")
	flag.BoolVar(&writeDot, "dot", writeDot, "Write graphviz file to stdout and exit")
	flag.Parse()
	if *fLog != "" {
		var lvv slog.LevelVar
		err := lvv.UnmarshalText([]byte(*fLog))
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		slog.SetLogLoggerLevel(lvv.Level())
	}
}

func main() {
	flags()

	prj := gomk.NewProject("")

	goalGoGen := prj.Goal(gomk.Abstract("go-gen")).
		By(&goGenerate)

	goalGovulnchk := prj.Goal(gomk.Abstract("vulncheck")).
		By(&goVulnchk)

	goalTest := prj.Goal(gomk.Abstract("test")).
		By(&goTest, goalGoGen, goalGovulnchk)

	goalBuildFoo := prj.Goal(gomk.File("cmd/foo/foo")).
		By(&goBuild, goalTest)

	goalBuildBar := prj.Goal(gomk.File("cmd/bar/bar")).
		By(&goBuild, goalTest)

	prj.Goal(gomk.Directory("dist")).
		By(gomk.FsCopy{MkDirs: true}, goalBuildFoo, goalBuildBar)

	if writeDot {
		prj.WriteDot(os.Stdout)
		return
	}

	builder := gomk.Builder{Env: gomk.DefaultEnv()}
	if flag.NArg() == 0 {
		if err := builder.Project(prj); err != nil {
			slog.Error(err.Error())
		}
	} else {
		if err := builder.NamedGoals(prj, flag.Args()...); err != nil {
			slog.Error(err.Error())
		}
	}
}
