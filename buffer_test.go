
package main

import (
	"testing"
	"strings"
	// XXX MULTI_CALLBACK
	"slices"
	"time"
	//"strconv"
	"fmt"
	"sync"

	"github.com/stretchr/testify/assert"
)



func TestAppendTrim(t *testing.T){
	buf := LinesBuffer{}

	i := buf.Append("a", "b", "c")

	assert.Equal(t, i, 2, 
		".Append(..): wrong length, got: %v expected: %v", i, 2)

	buf.Append(4, 5, 6)

	assert.Equal(t, len(buf.Lines), buf.Length, 
		".Append(..): wrong length, got: %v", len(buf.Lines))

	buf.Clear()

	assert.Equal(t, buf.Length, 0, 
		".Clear(): length not 0: %v", buf.Length)

	buf.Append("1\n2")
	buf.Append(3)

	assert.Equal(t, buf.Length, 3, 
		".Append(..): wrong length, got: %v", buf.Length)

	buf.Trim()

	assert.Equal(t, buf.Length, len(buf.Lines), 
		".Trim(): wrong length, got: %v", buf.Length)

	buf.Clear()

	// XXX need a better test...
	wg := sync.WaitGroup{}
	lines := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(){
			buf.Append("a", "b", "c")
			buf.Append(4, 5, 6)
			lines += 6
			wg.Done() }() }

	wg.Wait()

	assert.Equal(t, buf.Length, lines, 
		".Append(..): async: wrong length, got: %v expected: %v", buf.Length, lines)

	//fmt.Println("---", buf.String())
}
func TestBase(t *testing.T){
	buf := LinesBuffer{}

	t.Run("String", func(t *testing.T){
		assert.Equal(t, buf.String(), "", "Initial .String() failed")
	})

	s := "a line"

	// XXX this is the same as Append(..) test -- generalize...
	t.Run("Push", func(t *testing.T){
		buf.Append(s)
		assert.Equal(t, buf.String(), s, "Push failed")
		buf.Append(s)
		assert.Equal(t, buf.String(), s +"\n"+ s, "Push failed")
		buf.Append(s, s)
		assert.Equal(t, buf.String(), strings.Join([]string{s,s,s,s}, "\n"), "Push failed")
		buf.Append()
		assert.Equal(t, buf.String(), strings.Join([]string{s,s,s,s}, "\n"), "Push failed")
	})

	buf = LinesBuffer{}
	// XXX test reader/scanner...
	t.Run("Append", func(t *testing.T){
		buf.Append(s)
		assert.Equal(t, buf.String(), s, "Append failed")
		buf.Append(s)
		assert.Equal(t, buf.String(), s +"\n"+ s, "Append failed")
		buf.Append(s, s)
		assert.Equal(t, buf.String(), strings.Join([]string{s,s,s,s}, "\n"), "Append failed")
		buf.Append()
		assert.Equal(t, buf.String(), strings.Join([]string{s,s,s,s}, "\n"), "Append failed")
	})

	t.Run("Clear", func(t *testing.T){
		buf.Clear()
		assert.Equal(t, buf.String(), "", "Clear failed")
	})

	/* XXX
	t.Run("Write", func(t *testing.T){
		buf.Clear()
		buf.Append(s, s, s)
		assert.Equal(t, buf.String(), strings.Join([]string{s,s,s}, "\n"), "Append failed")
		buf.Write(s)
		assert.Equal(t, buf.String(), s, "Write failed")
	})
	//*/
}



//
//	.Transform(transformer[, mode])
//
//	mode:
//		"clear"
//
// XXX move to .Transform(..)
//		- start multiple handlers -- DONE
//		- stop/restart
//		- cleanup
// XXX BUG: Event: we still sometimes stall on .Changed.Wait() -- RACE???
// XXX TODO:
//		- .Trim() -- remove .Populated == false and trim to .Length
//		- restartable on .Append(..) -- DONE / XXX TEST
//		- resettable on .Write(..) -- DONE / XXX TEST
// NOTE: we do not care about callback(..) call order here -- sequencing 
//		callback(..) calls should be done by transformer(..)
func (this *LinesBuffer) Map(transformer Transformer, mode ...string) *LinesBuffer {

	// XXX do we need this???
	this.Transformers = append(this.Transformers, transformer)
	level := len(this.Transformers)

	i := 0
	to := 0
	seen := -1
	callback := func(from int) (func(string)){
		return func(s string){
			this.__writing.Lock()
			defer this.__writing.Unlock() 
			defer this.Changed.Trigger() 

			// handle inserts/shifts done by higher level transforms...
			for len(this.Lines) > to && 
					this.Lines[to].Transformed >= level {
				i++
				from++
				to++ }

			// handle inserts...
			// XXX do we need to handle appends separately???
			if seen == from {
				this.Lines = slices.Insert(this.Lines, to, Row{
					Transformed: -level,
					Populated: false,
				}) 
				i++ }
			seen = from

			// handle skips -- mark the skipped items as not printable...
			if from != to {
				for i := to+1; i <= from; i++ {
					this.Lines[i].Populated = false } }

			// update the row...
			this.Lines[to].Text = s
			this.Lines[to].Populated = true
			this.Lines[to].Transformed = level 
			to++ } }

	// restart...
	this.Cleared.On(
		func(){
			i = 0
			to = 0
			seen = -1 })

	// feed this.Lines to transformer(..)
	go func(){
		// transform (infinite loop)...
		for ; true; i++ {
			// handle trim...
			// XXX revise / test...
			if len(this.Lines) < i {
				i = len(this.Lines) -i
				to = i
				seen = i-1 }

			// wait till a new value is available...
			// NOTE: this handles appends...
			for i >= len(this.Lines) ||
					this.Lines[i].Transformed < level-1 {
				//this.Changed.Trigger()
				this.Changed.Wait() }

			row := &this.Lines[i]
			// mark row as read but not yet transformed...
			row.Transformed = -level
			// clear items before transform...
			if len(mode) > 0 && 
					mode[0] == "clear" {
				this.Lines[i].Populated = false }
			// NOTE: if transformer(..) calls callback(..) multiple times 
			//		it will update i...
			transformer(row.Text, callback(i)) } }()

	return this }
// Like .Map(..) but all Rows not processed yet are .Populated = false, 
// i.e. will not be returned by ..String()...
func (this *LinesBuffer) FMap(transformer Transformer, mode ...string) *LinesBuffer {
	return this.Map(transformer, "clear") }






func TestEvent(t *testing.T){
	evt := Event{}

	str := ""

	go func(){
		evt.Wait() 
		str += "A" }()

	time.Sleep(time.Millisecond*50)

	go func(){
		evt.Wait()
		str += "B"
		evt.Wait()
		str += "C" }()

	assert.Equal(t, "", str, "str must be empty.")

	// allow both goroutines reach .Wait()
	time.Sleep(time.Millisecond*100)

	evt.Trigger()
	time.Sleep(time.Millisecond*100)
	assert.Equal(t, "AB", str, "")

	evt.Trigger()
	time.Sleep(time.Millisecond*100)
	assert.Equal(t, "ABC", str, "")
}

// XXX make the tests programmatic...
func TestTransform(t *testing.T){
	buf := LinesBuffer{}

	buf.Write([]byte(
`one
two
three
four
five
six`))

	fmt.Println(buf.String())

	// append " a"
	buf.Map(
		func(s string, res TransformerCallback) {
			time.Sleep(time.Millisecond*100)
			//fmt.Println("   a:", s)
			res(fmt.Sprint(s, " a")) })

	// skip "three" + append " b"
	buf.Map(
		func(s string, res TransformerCallback) {
			// skip "three .."
			//fmt.Println("   b:", s)
			if strings.HasPrefix(s, "three") {
				return }
			res(fmt.Sprint(s, " b")) })

	// append " c" + append "new" after "two .."
	buf.Map(
		func(s string, res TransformerCallback) {
			time.Sleep(time.Millisecond*500)
			//fmt.Println("   c:", s)
			res(fmt.Sprint(s, " c"))
			// append new item after "two"
			if strings.HasPrefix(s, "two") {
				fmt.Println("   c:", "new")
				res("  new c") } })

	// append " d" + skip everything after "five .."
	skip := false
	buf.FMap(
		func(s string, res TransformerCallback) {
			if skip || strings.HasPrefix(s, "five") {
				skip = true
				return }
			//fmt.Println("   d:", s)
			res(fmt.Sprint(s, " d")) })

	// append " end"
	buf.Map(
		func(s string, res TransformerCallback) {
			//fmt.Println("   end:", s)
			res(fmt.Sprint(s, " end")) })


	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Millisecond * 500)

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second)

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second * 2)

	fmt.Println("---\n"+ buf.String())

	fmt.Println("--- (TRIM)")
	buf.Trim()

	fmt.Println(buf.String())
}


// XXX test shifts before an insert...
// XXX

func TestTransformBasic(t *testing.T){
	buf := LinesBuffer{}

	buf.Write([]byte(
`one
two
three
four
five
six`))

	fmt.Println(buf.String())

	// append " a"
	buf.Map(
		func(s string, res TransformerCallback) {
			//fmt.Println("   a:", s)
			res(fmt.Sprint(s, " a")) })
	// append " b"
	buf.Map(
		func(s string, res TransformerCallback) {
			if strings.HasPrefix(s, "three") {
				time.Sleep(time.Millisecond * 1500) }
			//fmt.Println("   b:", s)
			res(fmt.Sprint(s, " b")) })
	// append " end"
	buf.Map(
		func(s string, res TransformerCallback) {
			//fmt.Println("   end:", s)
			res(fmt.Sprint(s, " end")) })

	time.Sleep(time.Second)
	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second)
	fmt.Println("---\n"+ buf.String())

	buf.Append("appended")

	time.Sleep(time.Second)
	fmt.Println("---\n"+ buf.String())

	buf.Clear()

	time.Sleep(time.Second)
	fmt.Println("---\n"+ buf.String())

	buf.Append("one")
	buf.Append("more")
	buf.Append("line")

	time.Sleep(time.Second)
	fmt.Println("---\n"+ buf.String())
}


// XXX test shifts before an update...
func TestTransform2(t *testing.T){
	buf := LinesBuffer{}

	buf.Write([]byte(
`one
two
three
four
five
six`))

	fmt.Println(buf.String())

	// append " a"
	// XXX BUG: for some reason this sometimes gets applied for the second time...
	buf.Map(
		func(s string, res TransformerCallback) {
			if strings.HasPrefix(s, "three") {
				//fmt.Println("     SLEEP", s)
				time.Sleep(time.Second) 
				/*fmt.Println("     WAKE", s)*/ }
			//fmt.Println("   a:", s)
			res(fmt.Sprint(s, " a")) })

	//* XXX this breaks things quite badly...
	// XXX Problems:
	//		- stage 1: "three a" overwrites "two a"
	//			...either offsets not updated or insert not accounted for...
	//		- stage 1 sees items added on stage 2...
	//			...i.e. after WAKE stage 1 processes: "three", then "(two a)"
	// append " b" + insert "  ---"
	buf.Map(
		func(s string, res TransformerCallback) {
			//fmt.Println("   b:", s)
			res(fmt.Sprint(s, " b"))
			res(fmt.Sprint(" ("+ s +")")) })
	//*/

	// append " end"
	// XXX BUG? for some reason this does not get called for "five .." and onwards...
	// XXX BUG: sometimes the later handlers are called out of order...
	buf.Map(
		func(s string, res TransformerCallback) {
			//fmt.Println("   end:", s)
			res(fmt.Sprint(s, " end")) })

	fmt.Println("---")
	time.Sleep(time.Second * 2)

	fmt.Println(buf.String())

	fmt.Println("--- (TRIM)")
	buf.Trim()

	fmt.Println(buf.String())

	fmt.Println("--- (APPEND)")
	buf.Append("appended")

	time.Sleep(time.Second * 2)

	fmt.Println(buf.String())
}



// vim:set ts=4 sw=4 :
