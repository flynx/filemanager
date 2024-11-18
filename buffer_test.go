
package main

import (
	"testing"
	"strings"
	// XXX MULTI_CALLBACK
	//"slices"
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


//type TransformerCallback func(string)
//type Transformer func(string, TransformerCallback)

// XXX can't infer level from 0'th element as it can still be waiting 
//		for the last transform to write...
// XXX move to .Transform(..)
//		- start multiple handlers
//		- stop/restart
//		- cleanup
// XXX need to make this restartable...
func (this *LinesBuffer) _Transform(transformer Transformer) *LinesBuffer {

	// XXX do we need this???
	this.Transformers = append(this.Transformers, transformer)
	level := len(this.Transformers)

	// XXX set this once...
	if this.__wait == nil {
		this.__wait = make(chan bool) }

	// NOTE: we do not care about callback(..) call order here -- sequencing 
	//		callback(..) calls should be done by transformer(..)
	// XXX need to subtract skips from .length...
	//		...the problem is that we can count the skips once per-level 
	//		but need to subtract the skips from .Lines.length once per 
	//		skip (regardless of level)...
	to := 0
	length := this.Length
	callback := func(from int) (func(string)){
		return func(s string){
			// handle skips...
			if from != to {
				length = len(this.Lines) - (from - to) 
				// mark the skipped items as not printable...
				for i := to+1; i <= from; i++ {
					this.Lines[i].Populated = false } }

			// XXX handle reset...
			// XXX

			/* XXX MULTI_CALLBACK multiple calls to callback...
			// NOTE: we guard against inserts only as they can shift stuff 
			//		around and mess up indexes...
			inserting := false
			if this.__inserting.TryLock() {
				defer this.__inserting.Unlock()
			} else {
				inserting = true }
			//*/

			// XXX revise...
			//		...race?: can this be overwritten and cause a race?
			// XXX UGLY...
			defer func(){
				if this.__wait != nil {
					defer close(this.__wait) 
					this.__wait = make(chan bool) } }()
			if this.__wait == nil {
				this.__wait = make(chan bool) }

			/* XXX MULTI_CALLBACK multiple calls to callback...
			// handle inserts/shifts done by higher level transforms...
			for len(this.Lines) < to && 
					this.Lines[to].Transformed == level {
				to++ }
			// we outran the tail -> list truncated...
			// XXX should this be a panic???
			if len(this.Lines) < to {
				panic("list truncated...") }
			// append new elements...
			if len(this.Lines) == to {
				fmt.Println("!!! APPEND !!!")
				this.Lines = append(this.Lines, 
					Row{
						Transformed: -level,
						Populated: true,
					}) }
			// insert the new element...
			if this.Lines[to].Transformed != -level {
				fmt.Println("!!! INSERT !!!")
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
				//		once per transformer(..) call growing the output...
				// XXX ROWS
				this.Lines = slices.Insert(this.Lines, to, Row{
					Transformed: level-1,
					Populated: true,
				}) }
			//*/
			this.Lines[to].Text = s
			this.Lines[to].Populated = true
			this.Lines[to].Transformed = level 
			to++ } }

	// feed this.Lines to transformer(..)
	go func(){
		for i := 0; i < len(this.Lines); i++ {
			// XXX handle reset...
			// XXX
			row := &this.Lines[i]
			// wait till a new value is available...
			for row.Transformed < level-1 {
				<-this.__wait }
			// mark row as read but not yet transformed...
			row.Transformed = -level
			transformer(row.Text, callback(i)) } 
		// reflect skips...
		this.Length = length }()

	return this }
func (this *LinesBuffer) _String() string {
	rows := []string{}
	for _, row := range this.Lines {
		//if row.Transformed < len(this.Lines) {
		//	continue }
		rows = append(rows, row.Text) }
	return strings.Join(rows, "\n") }


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
		/* XXX 
		// append new item after "two"
		if strings.HasPrefix(s, "two") {
			res("new") }
		//*/
	})

	/*/ append " d"
	buf._Transform(func(s string, res TransformerCallback) {
		res(fmt.Sprint(s, " d"))
	})
	//*/

	//buf.transform()

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Millisecond * 500)

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second)

	fmt.Println("---\n"+ buf.String())

	time.Sleep(time.Second * 2)

	fmt.Println("---\n"+ buf.String())
}


// vim:set ts=4 sw=4 :
