package main

import (
	"fmt"

	"git.fractalqb.de/fractalqb/gomk/example/internal"
)

func main() {
	fmt.Printf("Red  : %d", internal.ColRed)
	fmt.Printf("Green: %d", internal.ColGreen)
	fmt.Printf("Blue : %d", internal.ColBlue)
}
