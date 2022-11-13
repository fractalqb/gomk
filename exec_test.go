package gomk

import (
	"context"
	"fmt"
	"time"
)

func ExampleCommand() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := Command(ctx, "echo", "Hello, command!")
	err := cmd.Run()
	fmt.Println(err)
	// Output:
	// Hello, command!
	// <nil>
}

func ExamplePipe() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pipe := BuildPipe(ctx).
		Command("echo", "Hello, pipe!").
		Command("cut", "-c", "8-11")
	err := pipe.Run()
	fmt.Println(err)
	// Output:
	// pipe
	// <nil>
}
