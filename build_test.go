package gomk

import (
	"fmt"
	"sort"
	"testing"
)

func TestEnv_zero(t *testing.T) {
	var e Env
	if e.CmdEnv() != nil {
		t.Error("unexpected zero env:", e.CmdEnv())
	}
}

func ExampleEnv() {
	var e Env
	e.Set("KEY1", "Just a value")
	fmt.Println(e.CmdEnv())
	e.Set("KEY2", "Yet another value")
	env := e.CmdEnv()
	sort.Strings(env)
	fmt.Println(env)
	fmt.Println(e.Get("KEY1"))
	e.Unset("KEY1")
	fmt.Println(e.CmdEnv())
	fmt.Println(e.Get("KEY1"))
	// Output:
	// [KEY1=Just a value]
	// [KEY1=Just a value KEY2=Yet another value]
	// Just a value true
	// [KEY2=Yet another value]
	//  false
}

func ExampleNewEnvOS() {
	e, _ := NewEnvOS([]string{
		"KEY1=Just a value",
		"KEY2=Yet another value",
	})
	fmt.Println(e.CmdEnv())
	// Output:
	// [KEY1=Just a value KEY2=Yet another value]
}
