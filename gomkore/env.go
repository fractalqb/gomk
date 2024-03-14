package gomkore

import (
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
)

type Env struct {
	In       io.Reader
	Out, Err io.Writer

	tags    map[string]string
	delt    map[string]bool
	xenv    []string
	xenvErr error
	parent  *Env
}

func DefaultEnv(tr *Trace) *Env {
	env := &Env{
		In:   os.Stdin,
		Out:  os.Stdout,
		Err:  os.Stderr,
		tags: make(map[string]string),
	}
	for _, evar := range os.Environ() {
		kv := strings.SplitN(evar, "=", 2)
		if len(kv) == 0 || kv[0] == "" {
			if tr != nil {
				tr.Warn("ignoring default `env`", `env`, evar)
			}
			continue
		}
		switch len(kv) {
		case 1:
			env.tags[kv[0]] = ""
		default:
			env.tags[kv[0]] = kv[1]
		}
	}
	return env
}

func (e *Env) Sub() *Env {
	return &Env{
		In: e.In, Out: e.Out, Err: e.Err,
		parent: e,
	}
}

func (e *Env) Clone() *Env {
	return &Env{
		In: e.In, Out: e.Out, Err: e.Err,
		tags: e.mergedTags(),
	}
}

func (e *Env) Tag(key string) (string, bool) {
	for e != nil {
		if e.tags != nil {
			if v, ok := e.tags[key]; ok {
				return v, true
			}
		}
		if e.delt != nil && e.delt[key] {
			break
		}
		e = e.parent
	}
	return "", false
}

func (e *Env) SetTag(key, val string) {
	if e.tags == nil {
		e.tags = make(map[string]string)
	}
	e.tags[key] = val
	if e.delt != nil {
		delete(e.delt, key)
	}
	e.clearXEnv()
}

func (e *Env) SetTags(env ...string) {
	if e.tags == nil {
		e.tags = make(map[string]string)
	}
	for _, evar := range env {
		kv := strings.SplitN(evar, "=", 2)
		switch len(kv) {
		case 1:
			e.tags[kv[0]] = ""
			if e.delt != nil {
				delete(e.delt, kv[0])
			}
		case 2:
			e.tags[kv[0]] = kv[1]
			if e.delt != nil {
				delete(e.delt, kv[0])
			}
		}
	}
	e.clearXEnv()
}

func (e *Env) SetTagsMap(tags map[string]string) {
	if e.tags == nil {
		e.tags = make(map[string]string)
	}
	maps.Copy(e.tags, tags)
	if e.delt != nil {
		for k := range tags {
			delete(e.delt, k)
		}
	}
	e.clearXEnv()
}

func (e *Env) DelTag(key string) {
	delete(e.tags, key)
	if e.parent != nil {
		if e.delt == nil {
			e.delt = make(map[string]bool)
		}
		e.delt[key] = true
	}
	e.clearXEnv()
}

type NonXEnvKeys []string

func (e NonXEnvKeys) Error() string {
	return fmt.Sprintf("illegal exec env keys: %s", strings.Join(e, ", "))
}

func (NonXEnvKeys) Is(target error) bool {
	_, ok := target.(NonXEnvKeys)
	return ok
}

func (e *Env) ExecEnv() ([]string, error) {
	if e.xenv == nil {
		var errKeys []string
		for k, v := range e.mergedTags() {
			switch {
			case k == "":
				errKeys = append(errKeys, `""`)
			case strings.ContainsRune(k, '='):
				errKeys = append(errKeys, k)
			default:
				tmp := fmt.Sprintf("%s=%s", k, v)
				e.xenv = append(e.xenv, tmp)
			}
		}
		if len(errKeys) > 0 {
			e.xenvErr = NonXEnvKeys(errKeys)
		}
	}
	return e.xenv, e.xenvErr
}

func (e *Env) clearXEnv() {
	e.xenv = nil
	e.xenvErr = nil
}

func (e *Env) mergedTags() map[string]string {
	if e.parent == nil {
		return maps.Clone(e.tags)
	}
	mts := e.parent.mergedTags()
	if e.delt != nil {
		for k := range e.delt {
			delete(mts, k)
		}
	}
	if e.tags != nil {
		maps.Copy(mts, e.tags)
	}
	return mts
}
