package gomk

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Build struct {
	PrjRoot string
	env     []string
}

func NewBuild(rootDir string) (*Build, error) {
	os.Environ()
	res := &Build{
		PrjRoot: rootDir,
	}
	if res.PrjRoot == "" {
		var err error
		if res.PrjRoot, err = os.Getwd(); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// func (b *Build) senv(key, val string) {
// 	prefix := key + "="
// 	for i, e := range b.env {
// 		if strings.HasPrefix(e, prefix) {
// 			b.env[i] = prefix + val
// 			return
// 		}
// 	}
// 	b.env = append(b.env, prefix+val)
// }

// func (b *Build) uenv(key string) {
// 	prefix := key + "="
// 	for i, e := range b.env {
// 		if strings.HasPrefix(e, prefix) {
// 			copy(b.env[i:], b.env[i+1:])
// 			b.env = b.env[:len(b.env)-1]
// 			return
// 		}
// 	}
// }

func (b *Build) WDir() *WDir {
	res := &WDir{b: b, dir: b.PrjRoot}
	res.pre = res
	return res
}

type WDir struct {
	b   *Build
	pre *WDir
	dir string
}

func (d *WDir) String() string {
	res := d.dir[len(d.b.PrjRoot):]
	if res == "" {
		return "/"
	}
	return res
}

func (d *WDir) Cd(dirs ...string) *WDir {
	tmp := make([]string, len(dirs)+1)
	tmp[0] = d.dir
	copy(tmp[1:], dirs)
	return &WDir{
		b:   d.b,
		pre: d,
		dir: filepath.Join(tmp...),
	}
}

func (d *WDir) Back() *WDir { return d.pre }

func (d *WDir) Do(what string, f func(dir *WDir)) {
	log.Printf("do in %s: %s\n", d, what)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d, what, p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d, what)
		}
	}()
	f(d)
}

func (d *WDir) Exec(exe string, args ...string) {
	cmd := exec.Command(exe, args...)
	log.Printf("exec in %s: %s\n", d, cmd)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d, cmd, p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d, cmd)
		}
	}()
	cmd.Dir = d.dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = d.b.env
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

type PipeError struct {
	act string
	cmd *exec.Cmd
	err error
}

func (pe PipeError) Unwrap() error { return pe.err }

func (pe PipeError) Error() string {
	return fmt.Sprintf("%s [%s]: %s", pe.act, pe.cmd, pe.err)
}

func (d *WDir) ExecPipe(cmds ...*exec.Cmd) {
	var sb strings.Builder
	l := len(cmds)
	if l == 0 {
		return
	}
	pipew := make([]*io.PipeWriter, 0, l-1)
	for i, cmd := range cmds {
		cmd.Dir = d.dir
		cmd.Stderr = os.Stderr
		cmd.Env = d.b.env
		if i > 0 {
			pr, pw := io.Pipe()
			pipew = append(pipew, pw)
			cmds[i-1].Stdout = pw
			cmd.Stdin = pr
			sb.WriteString(" | ")
		}
		fmt.Fprint(&sb, cmd.String())
	}
	log.Printf("pipe in %s: %s\n", d, sb.String())
	defer func() {
		if p := recover(); p != nil {
			log.Printf("failed in %s with %s: %s", d, sb.String(), p)
			panic(p)
		} else {
			log.Printf("done in %s with %s", d, sb.String())
		}
	}()
	cmds[0].Stdin = os.Stdin
	cmds[l-1].Stdout = os.Stdout
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			panic(PipeError{act: "start", cmd: cmd, err: err})
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			// TODO what about the rest .Wait() ?
			panic(PipeError{act: "wait", cmd: cmd, err: err})
		}
		if i < len(pipew) {
			pipew[i].Close()
		}
	}
}

func Try(f func()) (err error) {
	defer func() {
		if p := recover(); p != nil {
			switch e := p.(type) {
			case error:
				err = e
			case string:
				err = errors.New(e)
			default:
				err = fmt.Errorf("%+v", e)
			}
		}
	}()
	f()
	return nil
}
