package bees

import (
	"container/heap"
	"io"
	"runtime"
	"sync"
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

type stepHeap []*Step

func (sh stepHeap) Len() int { return len(sh) }

func (sh stepHeap) Less(i, j int) bool {
	si, sj := sh[i], sh[j]
	if si.depCount < 0 {
		return false
	} else if sj.depCount < 0 {
		return true
	}
	return si.depCount < sj.depCount
}

func (sh stepHeap) Swap(i, j int) {
	sh[i], sh[j] = sh[j], sh[i]
	sh[i].heapos = i
	sh[j].heapos = j
}

func (sh *stepHeap) Push(x interface{}) {
	step := x.(*Step)
	step.heapos = len(*sh)
	*sh = append(*sh, step)
}

func (sh *stepHeap) Pop() interface{} {
	lm1 := len(*sh) - 1
	res := (*sh)[lm1]
	*sh = (*sh)[:lm1]
	return res
}

func (sched *Scheduler) Update(s *Step, tag string, hive *Hive) (changed bool, err error) {
	log := qbsllm.New(
		sched.LogLevel,
		tag,
		sched.LogWriter,
		nil,
	)
	log.Infoa("running concurrent up-to-date phase from `step`", desc{s})
	var sheap stepHeap
	s.ForEach(func(s *Step) {
		s.changed = false
		s.depCount = len(s.deps)
		s.heapos = len(sheap)
		sheap = append(sheap, s)
	})
	heap.Init(&sheap)
	sched.makeHeap(hive, &sheap, log)
	return s.changed, nil
}

func (sched *Scheduler) makeHeap(hive *Hive, sthp *stepHeap, log *qbsllm.Logger) {
	var heapLock sync.Mutex
	heapChg := sync.NewCond(&heapLock)
	hive.start(log)
	go func() {
		for {
			heapLock.Lock()
			if sthp.Len() == 0 {
				heapLock.Unlock()
				break
			}
			for (*sthp)[0].depCount != 0 {
				heapChg.Wait()
			}
			next := heap.Pop(sthp).(*Step)
			heapLock.Unlock()
			hive.sched <- &job{step: next}
		}
		close(hive.sched)
	}()
	for job := range hive.respond {
		if job.res != nil {
			log.Debuga("clear schedule heap for `error` with `step`",
				job.res,
				desc{job.step})
			heapLock.Lock()
			*sthp = (*sthp)[:0]
			heapLock.Unlock()
			heapChg.Signal()
		} else {
			heapLock.Lock()
			for _, t := range job.step.tgts {
				t.changed = t.changed || job.step.changed
				t.depCount--
				heap.Fix(sthp, t.heapos)
			}
			heapLock.Unlock()
			if len(job.step.tgts) > 0 {
				heapChg.Signal()
			}
		}
	}
}

type job struct {
	step *Step
	res  error
}

type Hive struct {
	Bees    int
	size    int32
	sched   chan *job
	respond chan *job
	log     *qbsllm.Logger
}

func NewHive(size int) *Hive {
	if size < 1 {
		size = runtime.NumCPU() + size
		if size < 1 {
			size = 1
		}
	}
	return &Hive{Bees: size}
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

func (h *Hive) bee(id int) {
	log := h.log
	for job := range h.sched {
		step := job.step
		log.Tracea("`B`: -> `step`", id, desc{step})
		var (
			hint interface{}
			err  error
		)
		if step.changed {
			hint = DepChanged
			step.changed = false
		} else if step.UpToDate == nil {
			if len(step.deps) == 0 {
				hint = RootNode
			}
		} else {
			hint, err = step.UpToDate(job.step)
		}
		if err != nil {
			log.Errora("`B`: up-to-date check fails with `error`", id, err)
			job.res = err
			h.respond <- job
			continue
		}
		if hint == nil {
			log.Tracea("`B`: nothing to do for `step`", id, desc{job.step})
			job.res = nil
			h.respond <- job
			continue
		}
		if step.Build == nil {
			log.Infoa("`B`: non-build `step` (ignore `hint`)", id, desc{job.step}, hint)
			step.changed = true
		} else {
			log.Infoa("`B`: build `step` with `hint`", id, desc{job.step}, hint)
			step.changed, err = step.Build(job.step, hint)
		}
		if err != nil {
			log.Errora("`B`: build failed with `error`", id, err)
			job.res = err
		}
		h.respond <- job
	}
	if n := atomic.AddInt32(&h.size, -1); n == 0 {
		log.Debuga("`B`: last bee closes response channel", id)
		close(h.respond)
		h.sched = nil
		h.respond = nil
	} else if n < 0 {
		log.Errora("`B`: bee terminates with `count`", n)
	}
}

type desc struct {
	s *Step
}

func (d desc) String() string { return d.s.Description() }
