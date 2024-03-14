// This is an example gomk project that offers you a practical approach.
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	"git.fractalqb.de/fractalqb/eloc/must"
	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
)

var (
	// go generate ./...
	goGenerate = gomk.GoGenerate{FilesPkgs: []string{"./..."}}

	// go test ./...
	goTest = gomk.GoTest{Pkgs: []string{"./..."}}

	// go build -trimpath -s -w
	goBuild = gomk.GoBuild{TrimPath: true, LDFlags: []string{"-s", "-w"}}

	clean, dryrun bool
	writeDot      bool
	offline       bool

	tracr = gomk.WriteTracer{W: os.Stderr, Log: gomkore.TraceDebug}
)

func flags() {
	flag.BoolVar(&writeDot, "dot", writeDot, "Write graphviz file to stdout and exit")
	flag.BoolVar(&clean, "clean", clean, "Clean project")
	flag.BoolVar(&dryrun, "n", dryrun, "Dryrun")
	flag.BoolVar(&offline, "offline", offline, "Skip everything that requires being online")
	fLog := flag.String("log", "", "Set log level {off, warn, info, debug}")
	flag.Parse()

	if *fLog != "" {
		tracr.ParseLogFlag(*fLog)
	}
}

func main() {
	flags()

	prj := gomkore.NewProject("")

	must.Do(gomk.Edit(prj, func(prj gomk.ProjectEd) {
		goalGoGen, _ := prj.Goal(gomkore.Abstract("go-gen")).
			By(&goGenerate)

		goalTest, _ := prj.Goal(gomkore.Abstract("test")).
			By(&goTest, goalGoGen)

		goalPkgFoo := prj.Goal(mkfs.DirFiles("cmd/foo", "", 1)).
			ImpliedBy(goalTest)

		goalPkgBar := prj.Goal(mkfs.DirFiles("cmd/bar", "", 1)).
			ImpliedBy(goalTest)

		prj.Goal(mkfs.DirTree{
			Dir: "dist",
			Filter: mkfs.All{
				mkfs.IsDir(false),
				mkfs.Mode{Any: 0111},
				mkfs.MaxPathLen(1),
			},
		}).
			By(&goBuild, goalPkgFoo, goalPkgBar)

		docOutDir := mkfs.DirFiles("dist/doc", "", 0)

		mdSrcDir := mkfs.DirList{Dir: "doc", Filter: mkfs.NameMatch("*.md")}
		mdGoals := gomk.FsGoals(prj, mdSrcDir, nil)
		goals := gomk.Convert(mdGoals,
			gomk.OutFile{
				Ext:   gomk.ExtMap{".md": ".html"},
				Strip: mdSrcDir,
				Dest:  docOutDir,
			}.Artefact,
			func(out gomk.GoalEd) { out.SetRemovable(true) },
			// requires 'markdown' to be in the path
			&gomk.ConvertCmd{Exe: "markdown", PassOut: "stdout"},
			nil,
		)
		for _, g := range goals {
			g.SetRemovable(true)
		}
		goalDoc := prj.Goal(gomkore.Abstract("doc")).ImpliedBy(goals...)
		goalDoc.SetUpdateMode(gomk.UpdAllActions | gomk.UpdUnordered)

		pumlSrcDir := mkfs.DirList{Dir: "doc", Filter: mkfs.NameMatch("*.puml")}
		pumlGoals := gomk.FsGoals(prj, pumlSrcDir, nil)
		goals = gomk.Convert(pumlGoals,
			gomk.OutFile{
				Ext:   gomk.ExtMap{".puml": ".png"},
				Strip: pumlSrcDir,
				Dest:  docOutDir,
			}.Artefact,
			func(out gomk.GoalEd) { out.SetRemovable(true) },
			// requires 'plantuml' to be in the path
			&gomk.ConvertCmd{
				Exe:        "plantuml",
				PassOut:    "-o",
				OutRelToIn: true,
				OutDir:     true,
			},
			nil,
		)
		for _, g := range goals {
			g.SetRemovable(true)
		}
		goalDoc.ImpliedBy(goals...)
	}))

	tr := gomkore.NewTrace(context.Background(), tracr)

	if clean {
		err := gomkore.Clean(prj, dryrun, tr)
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

	builder := gomkore.Builder{} //LogDir: "build", MkDirMode: 0777}
	if flag.NArg() == 0 {
		if err := builder.Project(prj, tr); err != nil {
			slog.Error(err.Error())
		}
	} else {
		if err := builder.NamedGoals(prj, tr, flag.Args()...); err != nil {
			slog.Error(err.Error())
		}
	}
}
