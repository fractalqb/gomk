package run

import (
	"io"
	"os"

	"git.fractalqb.de/fractalqb/qbsllm"
)

type Logs interface {
	AllocLoc(stepId uintptr) *qbsllm.Logger
	ReleaseLoc(log *qbsllm.Logger)
	Stdout(stepId uintptr) io.Writer
	Stderr(stepId uintptr) io.Writer
}

type dLog int

const DefaultLogs dLog = 0

var (
	log    = qbsllm.New(qbsllm.Lnormal, "gomk", nil, nil)
	LogCgf = qbsllm.NewConfig(log)
)

func (_ dLog) AllocLoc(stepId uintptr) *qbsllm.Logger { return log }

func (_ dLog) ReleaseLoc(log *qbsllm.Logger) {}

func (_ dLog) Stdout(stepId uintptr) io.Writer { return os.Stdout }

func (_ dLog) Stderr(stepId uintptr) io.Writer { return os.Stderr }
