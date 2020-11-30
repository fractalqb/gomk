package bees

import (
	"fmt"
	"io"
	"strings"
	"unsafe"

	"git.fractalqb.de/fractalqb/icontainer/islist"
)

type Artefact interface {
	UpToDate(s *Step) (buildHint interface{}, err error)
}

type Builder interface {
	Build(s *Step, hint interface{}) (changed bool, err error)
}

type Describer interface {
	Describe(s *Step, w io.Writer)
}

// When to build a step:
//
// Dependencies \ UpToDate
// |           | is nil | eval: false | eval: true |
// |-----------+--------+-------------+------------|
// | len == 0  | build  | –           | build      |
// | changed   | build  | build       | build      |
// | no change | –      | –           | build      |
//   changed:   dependency was built and has changed in this build
//   no change: otherwise
type Step struct {
	subject       interface{}
	tgts, prereqs []*Step
	changed       bool // dual-use: 2nd during build when a dependency changed

	islist.NodeBase
	inslist  bool
	heapos   int
	depCount int
}

func NewStep(subject interface{}) *Step {
	res := &Step{
		subject: subject,
		heapos:  -1,
	}
	return res
}

func (s *Step) ID() uintptr {
	return uintptr(unsafe.Pointer(s))
}

func (s *Step) Description() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%x", s.ID())
	if desc, ok := s.subject.(Describer); ok {
		sb.WriteByte(':')
		desc.Describe(s, &sb)
	}
	sb.WriteByte(']')
	return sb.String()
}

func (s *Step) ForEach(do func(s *Step)) (n int) {
	lsOut := s.inslist
	var todo islist.List
	todo.PushBack(s)
	s.inslist = !lsOut
	for todo.Len() > 0 {
		next := todo.Front().(*Step)
		todo.Drop(1)
		do(next)
		n++
		for _, d := range next.prereqs {
			if d.inslist == lsOut {
				todo.PushBack(d)
				d.inslist = !lsOut
			}
		}
		for _, t := range next.tgts {
			if t.inslist == lsOut {
				todo.PushBack(t)
				t.inslist = !lsOut
			}
		}
	}
	return n
}

func (s *Step) AllPrereqs(do func(s *Step)) (n int) {
	lsOut := s.inslist
	var deps, clear islist.List
	defer func() {
		for c := deps.Front(); c != nil; c = c.ListNext() {
			c.(*Step).inslist = lsOut
		}
		for c := clear.Front(); c != nil; c = c.ListNext() {
			c.(*Step).inslist = lsOut
		}
	}()
	deps.PushBack(s)
	s.inslist = !lsOut
	for deps.Len() > 0 {
		next := deps.Front().(*Step)
		deps.Drop(1)
		clear.PushBack(next)
		do(next)
		n++
		for _, d := range next.prereqs {
			if d.inslist == lsOut {
				deps.PushBack(d)
				d.inslist = !lsOut
			}
		}
	}
	return n
}

func (s *Step) DependsOn(t *Step) bool {
	for _, d := range s.prereqs {
		if d == t {
			return true
		}
	}
	return false
}

func (s *Step) DependOn(ds ...*Step) *Step {
	for _, d := range ds {
		if !s.DependsOn(d) {
			s.prereqs = append(s.prereqs, d)
			d.tgts = append(d.tgts, s) // assumes consistency, no check
		}
	}
	return s
}

func (s *Step) Changed() bool { return s.changed }

type Done struct {
	s *Step
}

func (d Done) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Done %s", d.s.Description())
	return sb.String()
}
