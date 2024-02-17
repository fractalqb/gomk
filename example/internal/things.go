package internal

// Only online: go:generate go run golang.org/x/tools/cmd/stringer@latest -type Color
//
//go:generate stringer -type Color
type Color int

const (
	ColRed Color = iota
	ColGreen
	ColBlue
)
