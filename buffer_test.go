
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

	buf.Reset()

	assert.Equal(t, buf.Length, 0, 
		".Reset(): length not 0: %v", buf.Length)

	buf.Append("1\n2")
	buf.Append(3)

	assert.Equal(t, buf.Length, 3, 
		".Append(..): wrong length, got: %v", buf.Length)

	buf.Trim()

	assert.Equal(t, buf.Length, len(buf.Lines), 
		".Trim(): wrong length, got: %v", buf.Length)

	buf.Reset()

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


func (this *LinesBuffer) waitForTransform() *LinesBuffer {
	if this.__wait_transform == nil {
		this.__wait_transform = make(chan bool) }
	<-this.__wait_transform
	return this }
func (this *LinesBuffer) didTransform() *LinesBuffer {
	if this.__wait_transform != nil {
		defer close(this.__wait_transform) }
	this.__wait_transform = make(chan bool)
	return this }


// XXX move to .Transform(..)
//		- start multiple handlers -- DONE
//		- stop/restart
//		- cleanup
// XXX TODO:
//		- .Trim() -- remove .Populated == false and trim to .Length
//		- restartable on .Append(..)
//		- resettable on .Write(..)
func (this *LinesBuffer) _Transform(transformer Transformer) *LinesBuffer {

	// XXX do we need this???
	this.Transformers = append(this.Transformers, transformer)
	level := len(this.Transformers)

	// NOTE: we do not care about callback(..) call order here -- sequencing 
	//		callback(..) calls should be done by transformer(..)
	i := 0
	to := 0
	seen := -1
	length := this.Length
	callback := func(from int) (func(string)){
		return func(s string){
			this.__transforming.Lock()
			defer this.__transforming.Unlock() 

			// handle inserts/shifts done by higher level transforms...
			for len(this.Lines) < to && 
					this.Lines[to].Transformed == level {
				// XXX should we also inc from???
				i++
				from++
				to++ }

			// handle inserts...
			if seen == from {
				this.Lines = slices.Insert(this.Lines, to, Row{
					Transformed: -level,
					Populated: false,
				}) 
				i++
				this.Length++ }
			seen = from

			// handle skips...
			if from != to {
				length = len(this.Lines) - (from - to) 
				// mark the skipped items as not printable...
				for i := to+1; i <= from; i++ {
					this.Lines[i].Populated = false } }

			// update the row...
			this.Lines[to].Text = s
			this.Lines[to].Populated = true
			this.Lines[to].Transformed = level 
			to++ 

			this.didTransform() } }

	// feed this.Lines to transformer(..)
	go func(){
		for ; i < len(this.Lines); i++ {
			// XXX handle reset...
			// XXX
			row := &this.Lines[i]
			// wait till a new value is available...
			for row.Transformed < level-1 {
				this.waitForTransform() }
			// mark row as read but not yet transformed...
			row.Transformed = -level
			// NOTE: if transformer(..) calls callback(..) multiple times 
			//		it will update i...
			transformer(row.Text, callback(i)) } 
		// reflect skips...
		this.Length = length 
		// in case something is still waiting...
		this.didTransform() }()

	return this }


// XXX make the tests programmatic...
func TestTransformLocks(t *testing.T){
	buf := LinesBuffer{}

	go func(){
		buf.waitForTransform() 
		fmt.Println("A") }()
	go func(){
		buf.waitForTransform()
		fmt.Println("B") 
		buf.waitForTransform()
		fmt.Println("BB") }()

	time.Sleep(time.Millisecond*100)

	buf.didTransform()

	time.Sleep(time.Millisecond*100)

	buf.didTransform()

	time.Sleep(time.Millisecond*100)
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
	buf._Transform(func(s string, res TransformerCallback) {
		time.Sleep(time.Millisecond*100)
		fmt.Println("   a:", s)
		res(fmt.Sprint(s, " a"))
	})

	// skip "three" + append " b"
	buf._Transform(func(s string, res TransformerCallback) {
		// skip "three .."
		fmt.Println("   b:", s)
		if strings.HasPrefix(s, "three") {
			return }
		res(fmt.Sprint(s, " b"))
	})

	// append " c" + append "new" after "two .."
	buf._Transform(func(s string, res TransformerCallback) {
		time.Sleep(time.Millisecond*500)
		fmt.Println("   c:", s)
		res(fmt.Sprint(s, " c"))
		// append new item after "two"
		if strings.HasPrefix(s, "two") {
			fmt.Println("   c:", "new")
			res("new c") } })

	// append " d"
	buf._Transform(func(s string, res TransformerCallback) {
		res(fmt.Sprint(s, " d"))
	})

	// XXX test shifts before an update...
	// XXX

	// XXX test shifts before an insert...
	// XXX


	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Millisecond * 500)

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second)

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second * 2)

	fmt.Println("---\n"+ buf.String())
}


// vim:set ts=4 sw=4 :
