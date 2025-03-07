package gomk

import (
	"context"
	"os"
	"testing"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
	"git.fractalqb.de/fractalqb/testerr"
)

func TestGoals(t *testing.T) {
	prj := gomkore.NewProject(t.Name())
	g1 := testerr.Should1(prj.Goal(gomkore.Abstract("."))).BeNil(t)
	g2 := testerr.Should1(prj.Goal(mkfs.File("F"))).BeNil(t)
	gs := []*gomkore.Goal{g1, g2}

	t.Run("not exclusive", func(t *testing.T) {
		res := testerr.Shall1(Goals(gs, false, Tangible, AType[mkfs.File])).BeNil(t)
		if l := len(res); l != 1 {
			t.Fatalf("filter yields %d goals", l)
		}
		if res[0] != g2 {
			t.Fatalf("filtered wrong goal: %s", res[0])
		}
	})

	t.Run("exclusive good", func(t *testing.T) {
		res := testerr.Shall1(Goals(gs, true, Tangible, AType[mkfs.File])).BeNil(t)
		if l := len(res); l != 1 {
			t.Fatalf("filter yields %d goals", l)
		}
		if res[0] != g2 {
			t.Fatalf("filtered wrong goal: %s", res[0])
		}
	})

	t.Run("exclusive fail", func(t *testing.T) {
		testerr.Shall1(Goals(gs, true, Tangible, AType[mkfs.Directory])).
			Check(t, testerr.Msg("illegal goal 1: F"))
	})
}

func Test_buildProject(t *testing.T) {
	os.Remove("testdata/prj/doc/foo.cp")
	prj := gomkore.NewProject("testdata/prj")
	testerr.Shall(Edit(prj, func(prj ProjectEd) {
		prj.Goal(mkfs.File("doc/foo.cp")).
			By(mkfs.Copy{}, prj.Goal(mkfs.File("doc/foo.txt")))
	})).BeNil(t)
	build := NewBuilder(
		gomkore.NewTrace(context.Background(), TestTracer{t}),
		nil,
	)
	testerr.Shall(build.Project(prj)).BeNil(t)
	testerr.Shall1(os.Stat("testdata/prj/doc/foo.cp")).BeNil(t)
}
