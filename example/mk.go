package main

import (
	"flag"
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/gomk"
)

var (
	// As long as we don't have https://github.com/golang/go/issues/62418
	// create a logger that can change level.
	// (IMHO git.fractalqb.de/fractalqb/qblog has more readable output)
	logLevel slog.LevelVar
	log      = slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: &logLevel,
		},
	))

	// go generate ./...
	goGenerate = gomk.GoGenerate{FilesPkgs: []string{"./..."}}

	// go test ./...
	goTest = gomk.GoTest{Pkgs: []string{"./..."}}

	// govulncheck ./...
	goVulnchk = gomk.GoVulncheck{Patterns: []string{"./..."}}

	// go build -C <result-dir> -trimpath -s -w
	goBuild = gomk.GoBuild{
		TrimPath: true,
		LDFlags:  "-s -w",
	}

	writeDot bool
)

func flags() {
	fLog := flag.String("log", "", "Set log level")
	flag.BoolVar(&writeDot, "dot", writeDot, "Write graphviz file to stdout and exit")
	flag.Parse()
	if *fLog != "" {
		err := logLevel.UnmarshalText([]byte(*fLog))
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
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
	builder.Env.Log = log
	if flag.NArg() == 0 {
		if err := builder.Project(prj); err != nil {
			log.Error(err.Error())
		}
	} else {
		if err := builder.NamedGoals(prj, flag.Args()...); err != nil {
			log.Error(err.Error())
		}
	}
}
