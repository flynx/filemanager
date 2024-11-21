
package main

import (
	"testing"
	"strings"
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

	assert.Equal(t, len(buf.Lines), buf.Len(), 
		".Append(..): wrong length, got: %v", len(buf.Lines))

	buf.Clear()

	assert.Equal(t, buf.Len(), 0, 
		".Clear(): length not 0: %v", buf.Len())

	buf.Append("1\n2")
	buf.Append(3)

	assert.Equal(t, buf.Len(), 3, 
		".Append(..): wrong length, got: %v", buf.Len())

	buf.Trim()

	assert.Equal(t, buf.Len(), len(buf.Lines), 
		".Trim(): wrong length, got: %v", buf.Len())

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

	assert.Equal(t, buf.Len(), lines, 
		".Append(..): async: wrong length, got: %v expected: %v", buf.Len(), lines)

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


// XXX make the tests programmatic...
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

// XXX test shifts before an insert...
// XXX

// XXX insert still buggy -- looks like we either overwrite values or 
//		reaces on Event's .Wait() / .Trigger()...
// XXX make the tests programmatic...
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
	buf.Map(
		func(s string, res TransformerCallback) {
			if strings.HasPrefix(s, "three") {
				//fmt.Println("     SLEEP", s)
				time.Sleep(time.Second) 
				/*fmt.Println("     WAKE", s)*/ }
			//fmt.Println("   a:", s)
			res(fmt.Sprint(s, " a")) })

	// append " b" + insert "  ---"
	buf.Map(
		func(s string, res TransformerCallback) {
			//fmt.Println("   b:", s)
			res(fmt.Sprint(s, " b"))
			res(fmt.Sprint(" ("+ s +")")) })

	// append " end"
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

	fmt.Println("--- (CLEAR TRANSFORMS)")
	buf.ClearTransforms()
	// XXX BUG: this is written over the last item...
	buf.Append("appended clear")

	time.Sleep(time.Second * 2)

	fmt.Println(buf.String())
}



// vim:set ts=4 sw=4 :
