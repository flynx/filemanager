
package main

import (
	"testing"
	"strconv"
	"fmt"
)


func TestTeplateExpansion(t *testing.T){
	lines := Lines{}
	PLACEHOLDERS["TEST"] = 
		func(this *Lines, env Env) string {
			v, ok := env["TEST"]
			if ! ok {
				env["TEST"] = "1"
			} else {
				if i, err := strconv.Atoi(v); err == nil {
					env["TEST"] = fmt.Sprint(i+1) } }
			return env["TEST"] }

	n := 0
	test := func(tpl string, env Env, val string){
		t.Run(fmt.Sprintf("%v:%#v", n, tpl), func(t *testing.T){
			defer func(){ n++ }()
			res := lines.expandTemplate(tpl, env); 
			//if testing.Verbose() {
			//	//fmt.Printf("%v: .expandTemplate(%#v, %#v) -> %#v\n", n, tpl, env, res)
			//	fmt.Printf("%v: .expandTemplate(%#v, ...) -> %#v\n", n, tpl, res) }
			if res != val {
				t.Fatalf("%v: .expandTemplate(%#v):\n"+
					"\texpected: %#v\n"+
					"\tgot:      %#v",
					n, tpl, val, res) } }) }

	env := lines.makeEnv()

	// escaping...
	test("$$", env, "$")
	test("%%", env, "%")
	test("$$MOO", env, "$MOO")

	// default undefined values...
	test("$MOO", env, "")
	test("%MOO", env, "%MOO")

	// index...
	lines.Index = 0
	test("$INDEX", lines.makeEnv(), "0")
	lines.Index = 5
	test("$INDEX", lines.makeEnv(), "5")

	// line...
	lines.Index = 0
	test("$LINE", lines.makeEnv(), "1")
	lines.Index = 6
	test("$LINE", lines.makeEnv(), "7")

	// lines...
	lines.Clear()
	test("$LINES", lines.makeEnv(), "0")
	lines.Write("A\nB\nC")
	test("$LINES", lines.makeEnv(), "3")

	// %TEST
	env = lines.makeEnv()
	test("$TEST", env, "")
	test("%TEST", env, "1")
	test("%TEST", env, "2")
	test("%TEST", env, "3")
	test("$TEST", env, "3") 
	env["TEST"] = "0"
	test("$TEST", env, "0") 
}

func TestTeplateCMD(t *testing.T){
	// XXX
}

func TestParseSizes(t *testing.T){
	lines := Lines{}

	n := 0
	test := func(s string, w int, p int, expected []int){
		t.Run(fmt.Sprintf("%v:%#v:%v:%v", n, s, w, p), func(t *testing.T){
			defer func(){ n++ }()
			testSum := func(target string, l []int) bool {
				sum := 0
				c := 0
				for _, e := range l {
					if e >= 0 {
						c++
						sum += e } }
				sum += (c - 1) * p
				if sum != w {
					t.Fatalf("#%v: .parseSizes(%#v, %#v, %#v): bad "+target+" sum of widths:\n"+
							"\texpected sum must be: %#v\n"+
							"\tgot:                  %#v",
						n, s, w, p, w, sum) 
						return false } 
				return true }

			if ! testSum("expected", expected) {
				return }

			res := lines.parseSizes(s, w, p)

			//if testing.Verbose() {
			//	fmt.Printf("%v: .parseSizes(%#v, %v) -> %#v\n", n, s, w, res) }

			if ! testSum("resulting", res) {
				return }

			for i, v := range res {
				if len(expected) != len(res) || 
						expected[i] != v {
						t.Errorf("#%v: .parseSizes(%#v, %#v, %#v):\n"+
							"\texpected: %#v\n"+
							"\tgot:      %#v",
						n, s, w, p, expected, res) 
					return } } }) }

	// special case: single col...
	test("100%", 4, 0, []int{4})
	test("*", 4, 0, []int{4})
	test("", 4, 0, []int{4})

	test("50%", 100, 0, []int{50, 50})
	test("50%", 101, 0, []int{51, 50})
	test("50%", 101, 1, []int{50, 50})
	// XXX this yields an odd split of 49/51 -- can we make this more natural???
	//		...fixed but see note CEIL_ROUND
	//test("50%", 100, 1, []int{50, 49})
	test("50%", 100, 1, []int{49, 50})
	test("50%,", 101, 0, []int{51, 50})
	test("50%,50%", 100, 0, []int{50, 50})
	test("50%,50%", 101, 0, []int{51, 50})
	// XXX same as above note...
	//test("50%,50%", 100, 1, []int{50, 49})
	test("50%,50%", 100, 1, []int{49,50})
	test("50%,50%", 101, 1, []int{50,50})
	test("10,50%,10", 101, 0, []int{10,51,40})
	test("10,*,10", 101, 0, []int{10, 81, 10})
	test("10,*,*,10", 101, 0, []int{10, 41, 40, 10})
	test("*,*,*", 100, 0, []int{34, 33, 33})
	test("*,*,*", 20, 0, []int{7, 7, 6})
	test("*,*,*", 20, 1, []int{6,6,6})
	test("*,*,*,*", 20, 0, []int{5,5,5,5})

	// overflow...
	test("*,*,*,*,*,*", 20, 0, []int{5,5,5,5,-1,-1})
	test("*,*,*,*,*,*", 21, 0, []int{5,5,5,5,1,-1})
	test("*,*,*,*,*,*", 22, 0, []int{5,5,5,5,2,-1})
	test("*,*,*,*,*,*", 23, 0, []int{5,5,5,5,3,-1})
	test("*,*,*,*,*,*", 24, 0, []int{5,5,5,5,4,-1})
	test("*,*,*,*,*,*", 25, 0, []int{5,5,5,5,5,-1})
	test("*,*,*,*,*,*", 26, 0, []int{5,5,5,5,5,1})

	test("*,*,*,*,*,*", 19, 1, []int{4,4,4,4,-1,-1})
	test("*,*,*,*,*,*", 20, 1, []int{4,4,4,4,0,-1})
	test("*,*,*,*,*,*", 21, 1, []int{4,4,4,4,1,-1})
	test("*,*,*,*,*,*", 22, 1, []int{4,4,4,4,2,-1})
	test("*,*,*,*,*,*", 23, 1, []int{4,4,4,4,3,-1})
	test("*,*,*,*,*,*", 24, 1, []int{4,4,4,4,4,-1})
	test("*,*,*,*,*,*", 25, 1, []int{4,4,4,4,4,0})
	test("*,*,*,*,*,*", 26, 1, []int{4,4,4,4,4,1})

	// min overflow...
	// NOTE: the first col will get to min width and the seond will not fit...
	test("*,1", 4, 0, []int{4, -1})
	test("*,2", 4, 0, []int{4, -1})
	test("*,3", 4, 0, []int{4, -1})
	test("*,4", 4, 0, []int{4, -1})
	test("*,5", 4, 0, []int{4, -1})
	test("*,6", 4, 0, []int{4, -1})
	test("*,7", 4, 0, []int{4, -1})
	test("*,8", 4, 0, []int{4, -1})
	test("*,9", 4, 0, []int{4, -1})

	test("*,10", 9, 0, []int{5, 4}) 
}


func TestSection(t *testing.T){
	lines := Lines{}

	n := 0
	// XXX add overflow check...
	test := func(str string, w int) {
		t.Run(fmt.Sprintf("%v:%#v:%v", n, str, w), func(t *testing.T){
			defer func(){ n++ }()
			s, _ := lines.makeSection(str, w)
			//s, o := lines.makeSection(str, w)
			//if testing.Verbose() {
			//	fmt.Printf("%v: .makeSection(%#v, %v) -> %#v, %#v\n", n, str, w, s, o) }
			if len(s) != w {
				t.Fatalf("#%v: .makeSection(%#v, %#v): bad length::\n"+
						"\texpected: %#v\n"+
						"\tgot:      %#v",
					n, s, w, w, len(s)) } }) }

	test("underflow", 20)
	test("tab\ttab\ttab", 20)
	test("overflow overflow overflow overflow overflow overflow overflow", 20)
}

func TestSectionChrome(t *testing.T){
	// XXX
}

func TestDraw(t *testing.T){
	// XXX
}


