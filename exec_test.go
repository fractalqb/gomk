package gomk

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestPipe(t *testing.T) {
	pipe := PipeOp{
		CmdOp{Exe: "tr", Args: []string{"0123456789", "9876543210"}},
		CmdOp{Exe: "sort"},
	}
	var out strings.Builder
	env := Env{
		In:  strings.NewReader("1234\n4711\n"),
		Out: &out,
		Err: os.Stderr,
		Log: slog.Default(),
	}
	err := pipe.Do(context.Background(), nil, &env)
	if err != nil {
		t.Error(err)
	}
	if s := out.String(); s != "5288\n8765\n" {
		t.Errorf("bad output '%s'", s)
	}
}
