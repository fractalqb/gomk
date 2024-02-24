// Package gomkore implements the core model of gomk for the representation of
// buildable projects. It uses idiomatic Go error handling, which can make
// writing gomk build scripts a bit cumbersome. However, this package serves as
// a solid foundation for implementing build strategies, such as updating
// artifacts depending on their prerequisites or propagating artifact changes to
// dependent artifacts. The core concepts are [Project], [Goal] and [Action]. An
// easy-to-use wrapper for everyday use in build scripts is provided by the
// [gomk] package.
//
// [gomk]: https://pkg.go.dev/git.fractalqb.de/fractalqb/gomk
package gomkore
