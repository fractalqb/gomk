package gomk

import (
	"context"
	"fmt"
	"os"
	"time"
)

func ExamplePipe() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pipe := BuildPipeContext(ctx, nil).
		Command(nil, "echo", "Hello, pipe!").
		Command(nil, "cut", "-c", "8-11").
		SetStdout(os.Stdout)
	err := pipe.Run()
	fmt.Println(err)
	// Output:
	// pipe
	// <nil>
}
