package gomk

import (
	"testing"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
	"git.fractalqb.de/fractalqb/gomk/mkfs"
	"git.fractalqb.de/fractalqb/testerr"
)

func TestGoals(t *testing.T) {
	prj := gomkore.NewProject(t.Name())
	g1 := testerr.F1(prj.Goal(gomkore.Abstract("."))).ShouldBeNil(t)
	g2 := testerr.F1(prj.Goal(mkfs.File("F"))).ShouldBeNil(t)
	gs := []*gomkore.Goal{g1, g2}

	t.Run("not exclusive", func(t *testing.T) {
		res := testerr.F1(Goals(gs, false, Tangible, AType[mkfs.File])).ShallBeNil(t)
		if l := len(res); l != 1 {
			t.Fatalf("filter yields %d goals", l)
		}
		if res[0] != g2 {
			t.Fatalf("filtered wrong goal: %s", res[0])
		}
	})

	t.Run("exclusive good", func(t *testing.T) {
		res := testerr.F1(Goals(gs, true, Tangible, AType[mkfs.File])).ShallBeNil(t)
		if l := len(res); l != 1 {
			t.Fatalf("filter yields %d goals", l)
		}
		if res[0] != g2 {
			t.Fatalf("filtered wrong goal: %s", res[0])
		}
	})

	t.Run("exclusive fail", func(t *testing.T) {
		testerr.F1(Goals(gs, true, Tangible, AType[mkfs.Directory])).
			ShallMsg(t, "illegal goal 1: F")
	})
}
