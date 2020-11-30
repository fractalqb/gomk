package artifacts

import (
	"time"

	"git.fractalqb.de/fractalqb/gomk/bees"
)

type Timed interface {
	UpdateTime() time.Time
}

type TimedFile struct {
	name string
	t    time.Time
}

func (f *TimedFile) UpdateTime() time.Time { return f.t }

func (f *TimedFile) UpToDate(s *bees.Step) (buildHint interface{}, err error) {

}
