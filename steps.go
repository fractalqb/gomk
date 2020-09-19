package gomk

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
	update     uint32
	changed    bool

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

func (s *Step) MaxUpdate() (max uint32) {
	s.ForEach(func(s *Step) {
		if s.update > max {
			max = s.update
		}
	})
	return max
}

func (s *Step) SetUpdate(update uint32) (n int) {
	return s.ForEach(func(s *Step) { s.update = update })
}

func (s *Step) Roots() (roots []*Step, n int) {
	n = s.ForEach(func(s *Step) {
		if len(s.deps) == 0 {
			roots = append(roots, s)
		}
	})
	return roots, n
}

func (s *Step) Leaves() (leaves []*Step, n int) {
	n = s.ForEach(func(s *Step) {
		if len(s.tgts) == 0 {
			leaves = append(leaves, s)
		}
	})
	return leaves, n
}

func (s *Step) ForRoots(do func(s *Step)) (n int) {
	return s.ForEach(func(s *Step) {
		if len(s.deps) == 0 {
			do(s)
		}
	})
}

func (s *Step) ForLeaves(do func(s *Step)) (n int) {
	return s.ForEach(func(s *Step) {
		if len(s.tgts) == 0 {
			do(s)
		}
	})
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

func (s *Step) ChangedFor(update uint32) bool {
	return s.update == update && s.changed
}

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

type VisitFunc func(s *Step) error

func (start *Step) Forward(update uint32, do VisitFunc) (blockUpdId uint32, err error) {
	var todo islist.List
	pushTargets := func() {
		for _, t := range start.tgts {
			if t.update < update {
				todo.PushBack(t)
			} else if t.update > blockUpdId {
				blockUpdId = t.update
			}
		}
	}
	pushTargets()
	for todo.Len() > 0 {
		start = todo.Front().(*Step)
		todo.Drop(1)
		start.update = update
		if err := do(start); err != nil {
			return blockUpdId, err
		}
		pushTargets()
	}
	return blockUpdId, nil
}
