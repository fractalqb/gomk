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

// 1 2
func ExampleScheduler_Update() {
	net := testBeesNet1()
	hive := Hive{Bees: 1}
	NewScheduler(qbsllm.Linfo).Update(net[0], strconv.Itoa(0), &hive)
	NewScheduler(qbsllm.Linfo).Update(net[1], strconv.Itoa(1), &hive)
	// for i, step := range net {
	// 	fmt.Printf("Running update from step %d", i)
	// 	NewScheduler(qbsllm.Linfo).Update(step, strconv.Itoa(i), &hive)
	// }

	// Output:
	// _
}
