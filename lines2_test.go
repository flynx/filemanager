
package main

import "testing"


func TestTeplateExpansion(t *testing.T){
	//lines := Lines{}
}

func TestParseSizes(t *testing.T){
	lines := Lines{}

	testSizes := func(s string, w int, p int, expected []int){
		testSum := func(target string, l []int) bool {
			sum := 0
			c := 0
			for _, e := range l {
				if e >= 0 {
					c++
					sum += e } }
			sum += (c - 1) * p
			if sum != w {
				t.Errorf("parseSizes(%#v, %#v, %#v): bad "+target+" sum of widths:\n"+
						"\texpected sum must be: %#v\n"+
						"\tgot:                  %#v",
					s, w, p, w, sum) 
					return false } 
			return true }

		if ! testSum("expected", expected) {
			return }

		res := lines.parseSizes(s, w, p)

		if ! testSum("resulting", res) {
			return }

		for i, v := range res {
			if len(expected) != len(res) || 
					expected[i] != v {
				t.Errorf("parseSizes(%#v, %#v, %#v):\n"+
						"\texpected: %#v\n"+
						"\tgot:      %#v",
					s, w, p, expected, res) 
				return } } }


	testSizes("50%", 100, 0, []int{50, 50})
	testSizes("50%", 101, 0, []int{51, 50})
	testSizes("50%", 101, 1, []int{50, 50})
	// XXX this yields an odd split of 49/51 -- can we make this more natural???
	//		...fixed but see note CEIL_ROUND
	//testSizes("50%", 100, 1, []int{49, 50})
	testSizes("50%", 100, 1, []int{50, 49})
	testSizes("50%,", 101, 0, []int{51, 50})
	testSizes("50%,50%", 100, 0, []int{50, 50})
	testSizes("50%,50%", 101, 0, []int{51, 50})
	// XXX same as above note...
	//testSizes("50%,50%", 100, 1, []int{49,50})
	testSizes("50%,50%", 100, 1, []int{50, 49})
	testSizes("50%,50%", 101, 1, []int{50,50})
	testSizes("10,50%,10", 101, 0, []int{10,51,40})
	// XXX
	//testSizes("10,*,10", 101, 0, []int{10, 82, 9})
	testSizes("10,*,10", 101, 0, []int{10, 81, 10})
	// XXX
	//testSizes("10,*,*,10", 101, 0, []int{10, 41, 41, 9})
	testSizes("10,*,*,10", 101, 0, []int{10, 41, 40, 10})
	// XXX
	//testSizes("*,*,*", 100, 0, []int{34, 34, 32})
	testSizes("*,*,*", 100, 0, []int{34, 33, 33})
	testSizes("*,*,*", 20, 0, []int{7, 7, 6})
	testSizes("*,*,*", 20, 1, []int{6,6,6})
	// XXX
	//testSizes("*,*,*,*", 20, 0, []int{6,6,6,2})
	testSizes("*,*,*,*", 20, 0, []int{5,5,5,5})

	testSizes("*,*,*,*,*,*", 20, 0, []int{5,5,5,5,-1,-1})
	testSizes("*,*,*,*,*,*", 21, 0, []int{5,5,5,5,1,-1})
	testSizes("*,*,*,*,*,*", 22, 0, []int{5,5,5,5,2,-1})
	testSizes("*,*,*,*,*,*", 23, 0, []int{5,5,5,5,3,-1})
	testSizes("*,*,*,*,*,*", 24, 0, []int{5,5,5,5,4,-1})
	testSizes("*,*,*,*,*,*", 25, 0, []int{5,5,5,5,5,-1})
	testSizes("*,*,*,*,*,*", 26, 0, []int{5,5,5,5,5,1})

	testSizes("*,*,*,*,*,*", 19, 1, []int{4,4,4,4,-1,-1})
	testSizes("*,*,*,*,*,*", 20, 1, []int{4,4,4,4,0,-1})
	testSizes("*,*,*,*,*,*", 21, 1, []int{4,4,4,4,1,-1})
	testSizes("*,*,*,*,*,*", 22, 1, []int{4,4,4,4,2,-1})
	testSizes("*,*,*,*,*,*", 23, 1, []int{4,4,4,4,3,-1})
	testSizes("*,*,*,*,*,*", 24, 1, []int{4,4,4,4,4,-1})
	testSizes("*,*,*,*,*,*", 25, 1, []int{4,4,4,4,4,0})
	testSizes("*,*,*,*,*,*", 26, 1, []int{4,4,4,4,4,1})

	testSizes("", 4, 0, []int{4, -1})

	// row overflow...
	// NOTE: the first col will get to min width and the seond will not fit...
	testSizes("*,1", 4, 0, []int{4, -1})
	testSizes("*,2", 4, 0, []int{4, -1})
	testSizes("*,3", 4, 0, []int{4, -1})
	testSizes("*,4", 4, 0, []int{4, -1})
	testSizes("*,5", 4, 0, []int{4, -1})
	testSizes("*,6", 4, 0, []int{4, -1})
	testSizes("*,7", 4, 0, []int{4, -1})
	testSizes("*,8", 4, 0, []int{4, -1})
	testSizes("*,9", 4, 0, []int{4, -1})

	testSizes("*,10", 9, 0, []int{5, 4})

}
