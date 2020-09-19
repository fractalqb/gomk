package gomk

import (
	"fmt"
	"io"

	"git.fractalqb.de/fractalqb/qbsllm"
)

type thing string

var (
	_ Builder = thing("")
)

func (t thing) Build(_ *Step, _ interface{}) (bool, error) {
	fmt.Printf("Thing: %s\n", t)
	return true, nil
}

func (t thing) Describe(s *Step, w io.Writer) {
	io.WriteString(w, string(t))
}

type arte struct {
	thing
}

func (a arte) UpToDate(s *Step) (buildHint interface{}, err error) {
	return "test always builds", nil
}

func ExampleTraverse() {
	a := func(n string) arte { return arte{thing: thing(n)} }
	mn := NewStep(a("multi")).DependOn(NewStep(a("multi.1")))
	root := NewStep(a("1")).DependOn(
		NewStep(a("1.1")),
		NewStep(a("1.2")).DependOn(
			NewStep(a("1.1.1")),
			mn,
			NewStep(a("1.1.2")),
		),
		NewStep(a("1.3")).DependOn(mn),
	)
	hive := Hive{Bees: 1}
	NewScheduler(qbsllm.Ltrace).Update(root, 1, &hive)
	// Output:
	// Thing: 1.1
	// Thing: 1.1.1
	// Thing: multi.1
	// Thing: multi
	// Thing: 1.1.2
	// Thing: 1.2
	// Thing: 1.3
	// Thing: 1
}
