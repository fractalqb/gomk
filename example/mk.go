// This is an example gomk project that offers you a practical approach.
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

var (
	// Operation: go generate ./...
	goGenerate = gomk.GoGenerate{FilesPkgs: []string{"./..."}}

	// Operation: go test ./...
	goTest = gomk.GoTest{Pkgs: []string{"./..."}}

	// Operation: go build -trimpath -s -w
	goBuild = gomk.GoBuild{TrimPath: true, LDFlags: []string{"-s", "-w"}}

	tracer = gomk.NewDefaultTracer()

	// Some options (See also: https://pkg.go.dev/codeberg.org/fractalqb/gomklib#GoModule)
	clean, dryrun bool
	writeDot      bool
	offline       bool
)

func flags() {
	flag.BoolVar(&writeDot, "dot", writeDot, "Write graphviz file to stdout and exit")
	flag.BoolVar(&clean, "clean", clean, "Clean project")
	flag.BoolVar(&dryrun, "n", dryrun, "Dryrun")
	flag.BoolVar(&offline, "offline", offline, "Skip everything that requires being online")
	fTrace := flag.String("trace", "", "Set trace level")
	flag.Parse()

	tracer.ParseLevelFlag(*fTrace)
}

func main() {
	flags()

	// The project in current working dir
	prj := gomkore.NewProject("")

	// Start editing project, recovering panics to errors
	err := gomk.Edit(prj, func(prj gomk.ProjectEd) {
		goalGoGen, _ := prj.AbstractGoal("go-gen").
			By(&goGenerate)

		goalTest, _ := prj.AbstractGoal("test").
			By(&goTest, goalGoGen)

		goalPkgFoo := prj.Goal(mkfs.DirFiles("cmd/foo", "", 1)).
			ImpliedBy(goalTest)

		goalPkgBar := prj.Goal(mkfs.DirFiles("cmd/bar", "", 1)).
			ImpliedBy(goalTest)

		exes, _ := prj.Goal(mkfs.DirList{ // executable files in ./dist
			Dir:    "dist",
			Filter: mkfs.All{mkfs.IsDir(false), mkfs.Mode{Any: 0111}},
		}).
			By(&goBuild, goalPkgFoo, goalPkgBar)
		exes.SetRemovable(true) // Clean is allowed to remove these

		docOutDir := mkfs.DirFiles("dist/doc", "", 0)

		mdSrcDir := mkfs.DirList{Dir: "doc", Filter: mkfs.NameMatch("*.md")}
		mdGoals := gomk.GoalEds(prj, mdSrcDir)
		goals := gomk.Convert(mdGoals,
			gomk.OutFile{
				Ext:   gomk.ExtMap{".md": ".html"},
				Strip: mdSrcDir,
				Dest:  docOutDir,
			}.Artefact,
			func(out gomk.GoalEd) { out.SetRemovable(true) },
			&gomk.ConvertCmd{ // requires 'markdown' to be in the path
				Exe:     "markdown",
				PassOut: "stdout",
				MkDirs:  mkfs.MkDirs{MkDirMode: 0777},
			},
			nil,
		)
		for _, g := range goals {
			g.SetRemovable(true)
		}
		goalDoc := prj.Goal(gomkore.Abstract("doc")).ImpliedBy(goals...)
		goalDoc.SetUpdateMode(gomk.UpdAllActions | gomk.UpdUnordered)

		pumlSrcDir := mkfs.DirList{Dir: "doc", Filter: mkfs.NameMatch("*.puml")}
		pumlGoals := gomk.GoalEds(prj, pumlSrcDir)
		goals = gomk.Convert(pumlGoals,
			gomk.OutFile{
				Ext:   gomk.ExtMap{".puml": ".png"},
				Strip: pumlSrcDir,
				Dest:  docOutDir,
			}.Artefact,
			func(out gomk.GoalEd) { out.SetRemovable(true) },
			&gomk.ConvertCmd{ // requires 'plantuml' to be in the path
				Exe:        "plantuml",
				PassOut:    "-o",
				OutRelToIn: true,
				OutDir:     true,
				MkDirs:     mkfs.MkDirs{MkDirMode: 0777},
			},
			nil,
		)
		for _, g := range goals {
			g.SetRemovable(true)
		}
		goalDoc.ImpliedBy(goals...)
	})
	if err != nil {
		log.Fatal("editing project:", err)
	}
	tr := gomkore.NewTrace(context.Background(), tracer)

	if clean {
		err := gomk.Clean(prj, dryrun, tr)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if writeDot {
		dia := gomk.Diagrammer{RankDir: "LR"}
		if err := dia.WriteDot(os.Stdout, prj); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		return
	}

	build := gomk.NewBuilder(tr, nil)
	if flag.NArg() == 0 {
		if err := build.Project(prj); err != nil {
			slog.Error(err.Error())
		}
	} else {
		if err := build.NamedGoals(prj, flag.Args()...); err != nil {
			slog.Error(err.Error())
		}
	}
}
