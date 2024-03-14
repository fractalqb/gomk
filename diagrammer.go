package gomk

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"git.fractalqb.de/fractalqb/gomk/gomkore"
)

type Diagrammer struct {
	RankDir string
}

func (dia *Diagrammer) WriteDot(w io.Writer, prj *gomkore.Project) (err error) {
	defer func() {
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			case string:
				err = errors.New(p)
			default:
				err = fmt.Errorf("panic: %+v", p)
			}
		}
	}()

	dia.startDot(w, prj)
	for _, g := range prj.Goals(nil) {
		dia.goal(w, g)
	}
	for _, a := range prj.Actions() {
		dia.action(w, a)
	}
	dia.endDot(w)
	return nil
}

func (dia *Diagrammer) startDot(w io.Writer, prj *gomkore.Project) {
	fmt.Fprintf(w, "digraph \"%s\" {\n", escDotID(prj.Name(nil)))
	if dia.RankDir != "" {
		fmt.Fprintf(w, "\trankdir=\"%s\"\n", escDotID(dia.RankDir))
	}
}

func (dia *Diagrammer) endDot(w io.Writer) {
	fmt.Fprintln(w, "}")
}

func (dia *Diagrammer) goal(w io.Writer, g *gomkore.Goal) {
	var style string
	if g.IsAbstract() {
		if len(g.ResultOf()) == 0 || len(g.PremiseOf()) == 0 {
			style = "dashed,bold"
		} else {
			style = "dashed"
		}
		fmt.Fprintf(w, "\t\"%p\" [shape=box,style=\"%s\",label=\"%s\"];\n",
			g,
			style,
			g.Name(),
		)
		return
	} else if len(g.ResultOf()) == 0 || len(g.PremiseOf()) == 0 {
		style = ",style=bold"
	}

	atfType := reflect.Indirect(reflect.ValueOf(g.Artefact)).Type().Name()

	var updMode string
	if len(g.ResultOf()) > 1 {
		switch g.UpdateMode.Actions() {
		case UpdOneAction:
			updMode = " 1"
		case UpdAnyAction:
			updMode = " ?"
		case UpdSomeActions:
			updMode = " *"
		case UpdAllActions:
			updMode = " !"
		}
	}

	fmt.Fprintf(w, "\t\"%p\" [shape=record%s,label=\"{%s%s|%s}\"];\n",
		g,
		style,
		atfType,
		updMode,
		g.Name(),
	)
}

func (dia *Diagrammer) action(w io.Writer, a *gomkore.Action) {
	toRes := func(res *gomkore.Goal, implicit bool) {
		if res.UpdateMode.Ordered() {
			i := slices.Index(res.ResultOf(), a)
			if implicit {
				fmt.Fprintf(w, "\t\"%p\" -> \"%p\" [style=dashed,label=\"%d\"];\n",
					a,
					res,
					i+1,
				)
			} else {
				fmt.Fprintf(w, "\t\"%p\" -> \"%p\" [label=\"%d\"];\n",
					a,
					res,
					i+1,
				)
			}
		} else if implicit {
			fmt.Fprintf(w, "\t\"%p\" -> \"%p\" [style=dashed];\n", a, res)
		} else {
			fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", a, res)
		}
	}

	if a.Op == nil {
		if len(a.Results()) > 1 || len(a.Premises()) > 1 {
			fmt.Fprintf(w, "\t\"%p\" [shape=point];\n", a)
			for _, pre := range a.Premises() {
				fmt.Fprintf(w, "\t\"%p\" -> \"%p\" [style=dashed, arrowhead=none];\n", pre, a)
			}
			for _, res := range a.Results() {
				toRes(res, true)
			}
		} else if len(a.Premises()) == 0 {
			fmt.Fprintf(w, "\t\"%p\" [shape=point];\n", a)
			toRes(a.Result(0), true)
		} else if res := a.Result(0); res.UpdateMode.Ordered() {
			i := slices.Index(res.ResultOf(), a)
			fmt.Fprintf(w, "\t\"%p\" -> \"%p\" [style=dashed,label=\"%d\"];\n",
				a.Premise(0),
				res,
				i+1,
			)
		} else {
			fmt.Fprintf(w, "\t\"%p\" -> \"%p\" [style=dashed];\n",
				a.Premise(0),
				res,
			)
		}
		return
	}

	if len(a.Premises()) == 0 {
		fmt.Fprintf(w,
			"\t\"%p\" [shape=box,style=\"rounded,bold\",label=\"%s\"];\n",
			a,
			escDotID(a.String()),
		)
	} else {
		fmt.Fprintf(w,
			"\t\"%p\" [shape=box,style=\"rounded\",label=\"%s\"];\n",
			a,
			escDotID(a.String()),
		)
	}
	for _, pre := range a.Premises() {
		fmt.Fprintf(w, "\t\"%p\" -> \"%p\";\n", pre, a)
	}
	for _, res := range a.Results() {
		toRes(res, false)
	}
}

func escDotID(id string) string {
	return strings.ReplaceAll(id, "\"", "\\\"")
}
