package gomk

import (
	"log"
)

type ErrStater interface {
	ErrState() error
}

type OnErrFunc func(ErrStater)

func CheckErrState(onErr OnErrFunc, errst ErrStater) {
	if onErr != nil {
		onErr(errst)
	}
}

func Must(es ErrStater) {
	if err := es.ErrState(); err != nil {
		panic(err)
	}
}

func LogMust(es ErrStater) {
	if err := es.ErrState(); err != nil {
		log.Panic(err)
	}
}
