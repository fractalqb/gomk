package gomk

import (
	"context"
	"errors"
	"io"
	"path/filepath"
)

type RunEnv struct {
	Dir      *Dir
	Env      EnvVars
	In       io.Reader
	Out, Err io.Writer
}

func CtxEnv(ctx context.Context) *RunEnv {
	tmp := ctx.Value(ctxValTag{})
	if tmp == nil {
		return nil
	}
	return tmp.(*RunEnv)
}

func (cv *RunEnv) SetIO(clear bool, in io.Reader, out, err io.Writer) *RunEnv {
	if in != nil || clear {
		cv.In = in
	}
	if out != nil || clear {
		cv.Out = out
	}
	if err != nil || clear {
		cv.Err = err
	}
	return cv
}

func SubContext(ctx context.Context, joinPath string, envSet map[string]string, envUnset ...string) (
	context.Context, *RunEnv, error,
) {
	cv := CtxEnv(ctx)
	if cv == nil {
		return nil, nil, errors.New("no parent context value")
	}
	env := make(map[string]string)
	for k, v := range cv.Env.kvm {
		env[k] = v
	}
	for k, v := range envSet {
		env[k] = v
	}
	for _, u := range envUnset {
		delete(env, u)
	}
	cv = &RunEnv{
		Dir: cv.Dir.Join(joinPath),
		Env: *NewEnvVars(env),
		In:  cv.In,
		Out: cv.Out,
		Err: cv.Err,
	}
	return context.WithValue(ctx, ctxValTag{}, cv), cv, nil
}

func MustCd(ctx context.Context, elem ...string) context.Context {
	res, _, err := SubContext(ctx, filepath.Join(elem...), nil)
	if err != nil {
		panic(err)
	}
	return res
}

type ctxValTag struct{}
