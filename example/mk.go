// This is an example gomk project that offers you a practical approach.
package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"git.fractalqb.de/fractalqb/eloc/must"
	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
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

	must.Do(gomk.Edit(prj, func(prj gomk.ProjectEd) {
		goalGoGen := prj.Goal(gomk.Abstract("go-gen")).
			By(&goGenerate)

		goalTest := prj.Goal(gomk.Abstract("test")).
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

		mdSrcDir := mkfs.DirList{Dir: "doc", Filter: mkfs.NameMatch("*.md")}
		mdGoals := gomk.FsGoals(prj, mdSrcDir, nil)

		docOut := mkfs.DirFiles("dist/doc", "", 0)
		goals := gomk.Convert(mdGoals,
			gomk.OutFile{
				Ext:   gomk.ExtMap{".md": ".html"},
				Strip: mdSrcDir,
				Dest:  docOut,
			}.Artefact,
			// requires 'markdown' to be in the path
			&gomk.ConvertCmd{Exe: "markdown", Output: "stdout"},
		)
		goalDoc := prj.Goal(gomk.Abstract("doc")).ImpliedBy(goals...)
		goalDoc.SetUpdateMode(gomk.UpdAllActions | gomk.UpdUnordered)

		pumlSrcDir := mkfs.DirList{Dir: "doc", Filter: mkfs.NameMatch("*.puml")}
		pumlGoals := gomk.FsGoals(prj, pumlSrcDir, nil)
		goals = gomk.Convert(pumlGoals,
			gomk.OutFile{
				Ext:   gomk.ExtMap{".puml": ".png"},
				Strip: pumlSrcDir,
				Dest:  docOut,
			}.Artefact,
			// requires 'plantuml' to be in the path
			&gomk.ConvertCmd{
				Exe: "plantuml",
				// PlantUML takes -o relative to input, not CWD
				Args: []string{"-o", filepath.Join("..", prj.RelPath(docOut.Path()))},
			},
		)
		goalDoc.ImpliedBy(goals...)
	}))

	if clean {
		log := qblog.New(&qblog.DefaultConfig)
		err := gomk.Clean(prj, dryrun, log.Logger)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
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
