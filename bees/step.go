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
	Desc       func(*Step, io.Writer)
	UpToDate   func(*Step) (interface{}, error)
	Build      func(*Step, interface{}) (changed bool, err error)
	tgts, deps []*Step
	changed    bool // dual-use: 2nd during build when a dependency changed

	islist.NodeBase
	inslist  bool
	heapos   int
	depCount int
}

func NewStep(s interface{}) *Step {
	res := new(Step)
	if desc, ok := s.(Describer); ok {
		res.Desc = desc.Describe
	}
	if artf, ok := s.(Artefact); ok {
		res.UpToDate = artf.UpToDate
	}
	if bldr, ok := s.(Builder); ok {
		res.Build = bldr.Build
	}
	return res
}

func (s *Step) ID() uintptr {
	return uintptr(unsafe.Pointer(s))
}

func (s *Step) Description() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%x", s.ID())
	if s.Desc != nil {
		sb.WriteByte(':')
		s.Desc(s, &sb)
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
		for _, d := range next.deps {
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

func (s *Step) AllDeps(do func(s *Step)) (n int) {
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
		for _, d := range next.deps {
			if d.inslist == lsOut {
				deps.PushBack(d)
				d.inslist = !lsOut
			}
		}
	}
	return n
}

func (s *Step) DependsOn(t *Step) bool {
	for _, d := range s.deps {
		if d == t {
			return true
		}
	}
	return false
}

func (s *Step) DependOn(ds ...*Step) *Step {
	for _, d := range ds {
		if !s.DependsOn(d) {
			s.deps = append(s.deps, d)
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
	fmt.Fprintf(&sb, "Done [%X]", d.s.ID())
	if d.s.Desc != nil {
		fmt.Fprint(&sb, ": ")
		d.s.Desc(d.s, &sb)
	}
	return sb.String()
}
