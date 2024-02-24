package gomkore

import (
	"testing"
)

func TestEnv_SetTags(t *testing.T) {
	var e Env
	e.SetTags("")
	if v, ok := e.Tag(""); !ok {
		t.Error("empty tag not set")
	} else if v != "" {
		t.Errorf("emty tag has value '%s'", v)
	}
	e.SetTags("foo")
	if v, ok := e.Tag("foo"); !ok {
		t.Error("tag 'foo' not set")
	} else if v != "" {
		t.Errorf("tag 'foo' has value '%s'", v)
	}
	e.SetTags("foo=bar")
	if v, ok := e.Tag("foo"); !ok {
		t.Error("tag 'foo' not set")
	} else if v != "bar" {
		t.Errorf("tag 'foo' has value '%s'", v)
	}
	e.SetTags("=bar")
	if v, ok := e.Tag(""); !ok {
		t.Error("empty tag not set")
	} else if v != "bar" {
		t.Errorf("emty tag has value '%s'", v)
	}
}
