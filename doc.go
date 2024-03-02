// Package gomk helps to write build scripts in Go for projects where just
// running 'go build' is not enough. Instead of using platform-specific tools, a
// build script written in Go can better ensure platform independence. gomk is
// built around the core concepts of [Project], [Goal] and [Action]. The details
// are described in sub package [gomkore]. This packge wraps the code model from
// [gomkore] with a user-friendly API for building projects (see [Edit]).
//
// gomk is just a Go library. Is can be used in any context of reasonable
// programming with Go. But gomk is not a comprehensive library for all
// types of project builds. It is focussed on the fundamentals of a build
// system. Specific applications shall be implemented in separate libraries,
// such as [gomk-lib].
//
// Nevertheless, a few conventions can be helpful. A build script is a Go
// executable. As such it cannot be used by other packages (using [plugins] is
// not considered, primarily for not generally being portable).
//
//	"mk.go" is the recommended file name for a build script
//
// The build scripts of a project must not collide with the rest of the code.
// Here are a few ideas for structuring the build scripts:
//
// # Simple Go project with source in the root directory
//
//	module/
//	├── bar.go
//	├── foo.go
//	├── go.mod
//	├── go.sum
//	└── mk
//	    └── mk.go
//
// Build with
//
//	module$ go run mk/mk.go
//
// # Go project without code in the root directory
//
//	module/
//	├── cmd
//	│   ├── bar
//	│   │   └── main.go
//	│   └── foo
//	│       └── main.go
//	├── explib
//	│   └── stuff.go
//	├── go.mod
//	├── go.sum
//	├── internal
//	│   └── lib.go
//	└── mk.go
//
// Build with
//
//	module$ go run mk.go
//
// # Editing Projects
//
//	TODO: gomkore vs. simple API through [gomk.Edit]
//
// [plugins]: https://golang.google.cn/pkg/plugin/
// [gomk-lib]: https://pkg.go.dev/codeberg.org/fractalqb/gomklib
package gomk
