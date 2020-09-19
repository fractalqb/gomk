package gomk

import (
	"fmt"
	"io"
	"sync/atomic"

	"git.fractalqb.de/fractalqb/qbsllm"
)

//go:generate stringer -type BuildHint
type BuildHint int

const (
	Build BuildHint = iota + 1
	RootNode
	DepChanged
)

type Scheduler struct {
	LogLevel  qbsllm.Level
	LogWriter io.Writer
}

func NewScheduler(logLevel qbsllm.Level) *Scheduler {
	return &Scheduler{
		LogLevel: logLevel,
	}
}

func (sched *Scheduler) Update1(s *Step, update uint32) (changed bool, err error) {
	log := qbsllm.New(
		sched.LogLevel,
		fmt.Sprintf("U%d", update),
		sched.LogWriter,
		nil,
	)
	_, err = s.Backward(update, false, func(s *Step) error {
		var buildHint interface{}
		if len(s.deps) == 0 {
			if s.UpToDate == nil {
				buildHint = RootNode
			} else if buildHint, err = s.UpToDate(s); err != nil {
				log.Errora("update check for `step` fails with `err`", desc{s}, err)
				return err
			}
		} else {
			for _, d := range s.deps {
				if d.ChangedFor(update) {
					buildHint = DepChanged
					break
				}
			}
			if buildHint == nil && s.UpToDate != nil {
				buildHint, err = s.UpToDate(s)
				if err != nil {
					log.Errora("update check for `step` fails with `err`", desc{s}, err)
					return err
				}
			}
		}
		if buildHint == nil {
			log.Debuga("`step` is up to date", desc{s})
			s.changed = false
		} else if s.Build != nil {
			log.Infoa("build `step` with `hint`", desc{s}, buildHint)
			s.changed, err = s.Build(s, buildHint)
			if err != nil {
				log.Errora("build `step` failed with `err`", desc{s}, err)
			}
		} else {
			log.Debuga("`state` changed without rebuild, `hint`", desc{s}, buildHint)
			s.changed = true
		}
		return err
	})
	return s.ChangedFor(update), err
}

func (sched *Scheduler) Update(s *Step, update uint32, hive *Hive) (changed bool, err error) {
	log := qbsllm.New(
		sched.LogLevel,
		fmt.Sprintf("U%d", update),
		sched.LogWriter,
		nil,
	)
	log.Infoa("running concurrent up-to-date phase from `step`", desc{s})
	hive.start(log)
	phase := make(chan struct{})
	go func() {
		for resp := range hive.respond {
			if resp.hint != nil {
				resp.step.changed = true
			}
		}
		close(phase)
	}()
	_, err = s.Backward(update, false, func(s *Step) error {
		if len(s.deps) == 0 {
			if s.UpToDate == nil {
				s.changed = true
				// TODO build hint
				log.Debuga("root `step` is considered changed", desc{s})
			} else {
				log.Tracea("update >- `root` for up-to-date check", desc{s})
				hive.sched <- &job{step: s}
			}
		} else if s.UpToDate == nil {
			s.changed = false
		} else {
			log.Tracea("update >- `step` for up-to-date check", desc{s})
			hive.sched <- &job{step: s}
		}
		return nil
	})
	close(hive.sched)
	<-phase
	log.Infoa("up-to-date phase for `step` done", desc{s})
	s.Backward(update, true, func(s *Step) error {
		if s.changed {
			log.Infoa("`step` no up to date", desc{s})
		}
		return nil
	})
	return s.ChangedFor(update), err
}

type job struct {
	step *Step
	hint interface{}
	res  error
}

type Hive struct {
	Bees    int
	size    int32
	sched   chan *job
	respond chan *job
	log     *qbsllm.Logger
}

func (h *Hive) start(log *qbsllm.Logger) {
	if h.Bees <= 0 {
		h.Bees = 1
	}
	// TODO check for old channels
	h.log = log
	h.sched = make(chan *job)
	h.respond = make(chan *job)
	for i := 0; i < h.Bees; i++ {
		atomic.AddInt32(&h.size, 1)
		go h.bee(i)
	}
}

// TODO support parallel processing without breaking step order
func (h *Hive) bee(id int) {
	for job := range h.sched {
		h.log.Tracea("`B`: -> `step` with `hint`", id, desc{job.step}, job.hint)
		if job.hint == nil {
			h.upToDate(job)
		} else {
			h.build(job)
		}
	}
	if n := atomic.AddInt32(&h.size, -1); n == 0 {
		h.log.Debuga("`B`: last bee closes response channel", id)
		close(h.respond)
		h.sched = nil
		h.respond = nil
	} else if n < 0 {
		h.log.Errora("`B`: bee terminates with `count`", n)
	}
}

func (h *Hive) upToDate(job *job) {
	h.log.Tracea("bee checks `step` for being up to date", desc{job.step})
	if job.step.UpToDate != nil {
		job.hint, job.res = job.step.UpToDate(job.step)
		h.respond <- job
	}
}

func (h *Hive) build(job *job) {
	h.log.Tracea("bee builds `step` with `hint`", desc{job.step}, job.hint)
	var err error
	// job.step.targetsDepCount(-1)
	job.step.changed, err = job.step.Build(job.step, job.hint)
	if err != nil {
		h.log.Errora("bee build `step` failed with `err`", desc{job.step}, err)
	} else {
		h.log.Tracea("bee build `step` `changed`", desc{job.step}, job.step.changed)
	}
}

type desc struct {
	s *Step
}

func (d desc) String() string { return d.s.Description() }
