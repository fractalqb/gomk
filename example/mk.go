// This is an example gomk project that offers you a practical approach.
package main

import (
	"flag"
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/qblog"
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

	clean, dryrun bool
	writeDot      bool
	offline       bool
)

func flags() {
	fLog := flag.String("log", "", "Set log level")
	flag.BoolVar(&writeDot, "dot", writeDot, "Write graphviz file to stdout and exit")
	flag.BoolVar(&clean, "clean", clean, "Clean project")
	flag.BoolVar(&dryrun, "n", dryrun, "Dryrun")
	flag.BoolVar(&offline, "offline", offline, "Skip everything that requires being online")
	flag.Parse()
	if *fLog != "" {
		qblog.DefaultConfig.ParseFlag(*fLog)
	}
}

func main() {
	flags()

	builder := gomk.Builder{} //LogDir: "build", MkDirMode: 0777}
	prj := gomk.NewProject("", &builder)

	goalGoGen := prj.Goal(gomk.Abstract("go-gen")).
		By(&goGenerate)

	var goalGovulnchk, goalTest *gomk.Goal
	if offline {
		goalTest = prj.Goal(gomk.Abstract("test")).
			By(&goTest, goalGoGen)
	} else {
		goalGovulnchk = prj.Goal(gomk.Abstract("vulncheck")).
			By(&goVulnchk)

		goalTest = prj.Goal(gomk.Abstract("test")).
			By(&goTest, goalGoGen, goalGovulnchk)
	}

	goalBuildFoo := prj.Goal(gomk.File("cmd/foo/foo")).
		By(&goBuild, goalTest)

	goalBuildBar := prj.Goal(gomk.File("cmd/bar/bar")).
		By(&goBuild, goalTest)

	// requires 'markdown' to be in the path
	gomk.FsConvert(prj, gomk.DirContent("doc"), "doc/*.md", gomk.FsConverter{
		OutExt: ".html",
		Converter: &gomk.ConvertCmd{
			Exe:    "markdown",
			Output: "stdout",
		},
	})
	// requires 'plantuml' to be in the path
	gomk.FsConvert(prj, gomk.DirContent("doc"), "doc/*.puml", gomk.FsConverter{
		OutExt:    ".png",
		Converter: &gomk.ConvertCmd{Exe: "plantuml"},
	})

	prj.Goal(gomk.DirList("dist")).
		By(gomk.FsCopy{MkDirMode: 0777}, goalBuildFoo, goalBuildBar)

	if writeDot {
		if _, err := prj.WriteDot(os.Stdout); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		return
	}

	if clean {
		log := qblog.New(&qblog.DefaultConfig)
		err := gomk.Clean(prj, dryrun, log.Logger)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		return
	}

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
