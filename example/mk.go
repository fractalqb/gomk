// This is an example gomk project that offers you a practical approach.
package main

import (
	"flag"
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/eloc/must"
	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/qblog"
)

var (
	// go generate ./...
	goGenerate = gomk.GoGenerate{FilesPkgs: []string{"./..."}}

	// go test ./...
	goTest = gomk.GoTest{Pkgs: []string{"./..."}}

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

	prj := gomk.NewProject("")

	must.Do(gomk.Edit(prj, func(prj gomk.ProjectEd) error {
		goalGoGen := prj.Goal(gomk.Abstract("go-gen")).
			By(&goGenerate)

		goalTest := prj.Goal(gomk.Abstract("test")).
			By(&goTest, goalGoGen)

		goalBuildFoo := prj.Goal(gomk.File("cmd/foo/foo")).
			By(&goBuild, goalTest)

		goalBuildBar := prj.Goal(gomk.File("cmd/bar/bar")).
			By(&goBuild, goalTest)

		mdGoals := must.Ret(gomk.FsGoals(prj, gomk.DirList{Dir: "doc", Glob: "*.md"}, nil))
		goals := gomk.Convert(mdGoals,
			gomk.FileExt(".html").Convert,
			// requires 'markdown' to be in the path
			&gomk.ConvertCmd{Exe: "markdown", Output: "stdout"},
		)
		goalDoc := prj.Goal(gomk.Abstract("doc")).ImpliedBy(goals...)

		pumlGoals := must.Ret(gomk.FsGoals(prj, gomk.DirList{Dir: "doc", Glob: "*.puml"}, nil))
		goals = gomk.Convert(pumlGoals,
			gomk.FileExt(".png").Convert,
			// requires 'plantuml' to be in the path
			&gomk.ConvertCmd{Exe: "plantuml"},
		)
		goalDoc.ImpliedBy(goals...)

		prj.Goal(gomk.DirList{Dir: "dist"}).
			By(gomk.FsCopy{MkDirMode: 0777}, goalBuildFoo, goalBuildBar)

		return nil
	}))

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

	builder := gomk.Builder{} //LogDir: "build", MkDirMode: 0777}
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
