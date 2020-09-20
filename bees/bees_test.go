package bees

import (
	"fmt"
	"io"
	"strconv"

	"git.fractalqb.de/fractalqb/qbsllm"
)

type testArtefact struct {
	desc string
}

func (ta *testArtefact) Describe(_ *Step, w io.Writer) {
	io.WriteString(w, ta.desc)
}

func (ta *testArtefact) Build(_ *Step, hint interface{}) (changed bool, err error) {
	fmt.Printf("build step %s with hint '%s'\n", ta.desc, hint)
	return true, nil
}

//  0 1
//  |\|
//  2 3
//  |\|
//  4 5
func testBeesNet1() (res []*Step) {
	for i := 0; i < 6; i++ {
		s := NewStep(&testArtefact{desc: strconv.Itoa(i)})
		res = append(res, s)
	}
	res[0].DependOn(res[2], res[3])
	res[1].DependOn(res[3])
	res[2].DependOn(res[4], res[5])
	res[3].DependOn(res[5])
	return res
}

func ExampleScheduler_Make() {
	net := testBeesNet1()
	hive := Hive{Bees: 1}
	sched := NewScheduler(qbsllm.Lwarn)
	for i, step := range net {
		fmt.Printf("Running update from step %d\n", i)
		sched.Make(step, strconv.Itoa(i), &hive)
	}
	// Output:
	// Running update from step 0
	// build step 4 with hint 'RootNode'
	// build step 5 with hint 'RootNode'
	// build step 2 with hint 'DepChanged'
	// build step 3 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// Running update from step 1
	// build step 5 with hint 'RootNode'
	// build step 4 with hint 'RootNode'
	// build step 3 with hint 'DepChanged'
	// build step 2 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
	// Running update from step 2
	// build step 4 with hint 'RootNode'
	// build step 5 with hint 'RootNode'
	// build step 2 with hint 'DepChanged'
	// build step 3 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// Running update from step 3
	// build step 5 with hint 'RootNode'
	// build step 4 with hint 'RootNode'
	// build step 3 with hint 'DepChanged'
	// build step 2 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
	// Running update from step 4
	// build step 4 with hint 'RootNode'
	// build step 5 with hint 'RootNode'
	// build step 2 with hint 'DepChanged'
	// build step 3 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// Running update from step 5
	// build step 5 with hint 'RootNode'
	// build step 4 with hint 'RootNode'
	// build step 3 with hint 'DepChanged'
	// build step 2 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
}

func ExampleScheduler_Update() {
	net := testBeesNet1()
	hive := Hive{Bees: 1}
	sched := NewScheduler(qbsllm.Lwarn)
	for i, step := range net {
		fmt.Printf("Running update from step %d\n", i)
		sched.Update(step, strconv.Itoa(i), &hive)
	}
	// Output:
	// Running update from step 0
	// build step 4 with hint 'RootNode'
	// build step 5 with hint 'RootNode'
	// build step 2 with hint 'DepChanged'
	// build step 3 with hint 'DepChanged'
	// build step 0 with hint 'DepChanged'
	// Running update from step 1
	// build step 5 with hint 'RootNode'
	// build step 3 with hint 'DepChanged'
	// build step 1 with hint 'DepChanged'
	// Running update from step 2
	// build step 4 with hint 'RootNode'
	// build step 5 with hint 'RootNode'
	// build step 2 with hint 'DepChanged'
	// Running update from step 3
	// build step 5 with hint 'RootNode'
	// build step 3 with hint 'DepChanged'
	// Running update from step 4
	// build step 4 with hint 'RootNode'
	// Running update from step 5
	// build step 5 with hint 'RootNode'
}
