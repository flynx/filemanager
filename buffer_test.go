
package main

import (
	"testing"
	"strings"
	"slices"
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


//type TransformerCallback func(string)
//type Transformer func(string, TransformerCallback)

// XXX need this to be able to auto-restart on rows reset...
type T struct {
	__wait chan bool
	__inserting sync.Mutex
}
func (this *T) Write(str string) (*T) {
	// XXX
	return this }
func (this *T) Wait() (*T) {
	<-this.__wait
	return this }
// XXX can't infer level from 0'th element as it can still be waiting 
//		for the last transform to write...
// XXX move to .Transform(..)
//		- start multiple handlers
//		- stop/restart
//		- cleanup
// XXX need to make this restartable...
func NewT(rows []Row, level int, f Transformer) *T {
	this := &T{}

	// XXX set this once...
	this.__wait = make(chan bool)

	to := 0
	// NOTE: we do not care about callback(..) call order here -- sequencing 
	//		callback(..) calls should be done by f(..)
	var callback func(string)
	callback = func(s string){
		// XXX handle reset...
		// XXX

		// NOTE: we guard against inserts only as they can shift stuff 
		//		around and mess up indexes...
		inserting := false
		if this.__inserting.TryLock() {
			defer this.__inserting.Unlock()
		} else {
			inserting = true }

		// XXX revise...
		//		...race?: can this be overwritten and cause a race?
		defer close(this.__wait)
		this.__wait = make(chan bool)

		// handle shifts done by higher level transforms...
		for len(rows) < to && 
				rows[to].Transformed == level {
			to++ }
		if len(rows) < to {
			// XXX we outran the tail -> list truncated...
			// XXX should this be a panic???
			panic("list truncated...") }
		// append new elements...
		if len(rows) == to {
			// XXX ROWS
			rows = append(rows, 
				Row{
					Transformed: -level,
					Populated: true,
				}) }
		// insert the new element...
		if rows[to].Transformed != -level {
			// wait for other inserts to finixh...
			if inserting {
				// NOTE: this is used to wait/block till other inserts 
				//		are done only, thus the immediate unlock...
				this.__inserting.Lock()
				this.__inserting.Unlock() 
				// NOTE: we can't trust indexes at this point so we need 
				//		to do a clean retry... (XXX)
				callback(s) 
				return }
			// NOTE: this can be needed if callback is called more than 
			//		once per f(..) call growing the output...
			// XXX ROWS
			rows = slices.Insert(rows, to, Row{
				Transformed: level-1,
				Populated: true,
			}) }
		rows[to].Text = s
		rows[to].Transformed = level 
		to++ }

	// feed rows to f(..)
	for i:=0 ; i < len(rows); i++ {
		// XXX handle reset...
		// XXX
		row := rows[i]
		// wait till a new value is available...
		for row.Transformed < level-1 {
			<-this.__wait }
		// mark row as read but not yet transformed...
		rows[i].Transformed = -level
		f(row.Text, callback) }

	return this }



func TestTransform(t *testing.T){
	buf := LinesBuffer{}

	buf.Write([]byte(
`one
two
three
four`))

	fmt.Println(buf.String())

	buf.Transform(func(s string, res TransformerCallback) {
		res(fmt.Sprint(s, " a"))
		// XXX append new item after "two"
		//if s == "two" {
		//	res("new")
		//}
	})
	buf.Transform(func(s string, res TransformerCallback) {
		// skip "three .."
		if strings.HasPrefix(s, "three") {
			return
		}
		res(fmt.Sprint(s, " b"))
	})
	buf.Transform(func(s string, res TransformerCallback) {
		res(fmt.Sprint(s, " c"))
	})

	buf.transform()

	fmt.Println(buf.String())
}


// vim:set ts=4 sw=4 :
