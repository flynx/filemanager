
package main

import (
	"fmt"
	//"log"
	"io"
	"strings"
	"slices"
	"bufio"
	"sync"
)



// XXX
type Transformer func(string)string


// LinesBuffer
//
type LinesBuffer struct {
	sync.Mutex
	Lines []Row
	Index int
	Length int

	// XXX
	Transformers []Transformer

	__writing sync.Mutex
	__transforming sync.Mutex
}
// Editing...
//
func (this *LinesBuffer) Clear() *LinesBuffer {
	this.Lines = []Row{}
	this.Index = 0
	this.Length = 0
	return this }

func (this *LinesBuffer) String() string {
	lines := []string{}
	for i, line := range this.Lines {
		if i >= this.Length {
			break }
		lines = append(lines, line.Text) }
	return strings.Join(lines, "\n") }

func (this *LinesBuffer) Reset() *LinesBuffer {
	this.Length = 0
	return this }
func (this *LinesBuffer) Trim() *LinesBuffer {
	this.__writing.Lock()
	this.Lines = this.Lines[:this.Length]
	this.__writing.Unlock()
	return this }

func (this *LinesBuffer) Append(strs ...any) int {
	this.__writing.Lock()
	i := this.Length
	l := 0
	// normalize inputs + count lines if possible...
	for i, in := range strs {
		switch in.(type) {
			// readers make things non-deterministic...
			case io.Reader:
				l = -1
				defer this.__writing.Unlock() 
			case []byte:
				lst := strings.Split(string(in.([]byte)), "\n")
				strs[i] = lst
				if l >= 0 {
					l += len(lst) }
			case string:
				lst := strings.Split(in.(string), "\n")
				strs[i] = lst
				if l >= 0 {
					l += len(lst) }
			case []string:
				if l >= 0 {
					l += len(in.([]string)) }
			// convert any to string...
			default:
				strs[i] = fmt.Sprint(in)
				if l >= 0 {
					l++ } } }
	// grow .Lines if needed...
	if l >= 0 {
		this.Length += l 
		if this.Length > len(this.Lines) {
			slices.Grow(this.Lines, this.Length - len(this.Lines)) }
		this.__writing.Unlock() }

	// append...
	//
	place := func(i int, s string) bool {
		// the list has been trimmed...
		if this.Length < i {
			return false }
		row := Row{ Text: s }
		if i < len(this.Lines) {
			this.Lines[i] = row
		// NOTE: since we are adding line-by-line there is not chance 
		//		that we are more than 1 off, unless we .Trim() while we 
		//		are running...
		} else {
			this.Lines = append(this.Lines, row) } 
		return true }
	for _, in := range strs {
		stop := false
		switch in.(type) {
			case io.Reader:
				scanner := bufio.NewScanner(in.(io.Reader))
				for ! stop && 
						scanner.Scan() {
					stop = ! place(i, scanner.Text()) 
					i++ }
			case []string:
				for _, in := range in.([]string) {
					if stop = ! place(i, in) ; stop {
						break } 
					i++ } 
			case string:
				if stop = ! place(i, in.(string)) ; stop {
					break } 
				i++ }
		if stop {
			break } }

	return i-1 }
// XXX do we need .Write(..)
// XXX HACK: no error handling...
func (this *LinesBuffer) Write(b []byte) (int, error) {
	this.Clear()
	this.Append(string(b))
	return len(b), nil }

// XXX not sure about the transformer API yet...
func (this *LinesBuffer) Transform(transformer Transformer) *LinesBuffer {
	//this.transformers = append(this.transformers, transformer)
	return this }
// XXX this should account for blocking transformers...
//		i.e. if a tranformer blocks it should block all the higher number
//		transformers from passing it while all lower number transformers 
//		should procede...
// XXX should the transformr stack be editable/viewable (api)???
// XXX needs a rethink....
func (this *LinesBuffer) triggerTransform() {
	if ! this.__transforming.TryLock() {
		return }
	defer this.__transforming.Unlock()

	wg := sync.WaitGroup{}
	prev := make(chan string)
	in := prev
	for level, transform := range this.Transformers {
		// XXX need to maintain write position per transformer...
		// XXX this should be extrnally resettable...
		i := 0
		in := prev
		out := make(chan string)
		prev = out
		wg.Add(1)
		go func(){
			defer wg.Done()
			for true {
				str, ok := <-in
				if ! ok {
					break }
				out <- transform(str)
				this.__writing.Lock()
				this.Lines[i].Text = <-out
				// XXX is this needed???
				this.Lines[i].Transformed = level
				this.__writing.Unlock() } }() }

	// feed the chain...
	for _, row := range this.Lines {
		in <- row.Text }
	wg.Wait() }

// High-level...
//
func (this *LinesBuffer) Current() string {
	if len(this.Lines) == 0 {
		return "" }
	return this.Lines[this.Index].Text }
func (this *LinesBuffer) SelectedRows() []Row {
	res := []Row{}
	for _, row := range this.Lines {
		if row.Selected {
			res = append(res, row) } }
	return res }
func (this *LinesBuffer) Selected() []string {
	res := []string{}
	for _, row := range this.Lines {
		if row.Selected {
			res = append(res, row.Text) } }
	return res }
// XXX would be nice to make this generic...
func (this *LinesBuffer) Select(selection any, mode ...Toggle) *LinesBuffer {
	var m Toggle 
	if len(mode) != 0 {
		m = mode[0] }

	toggle := func(lst []Row, i int){
		lst[i].Selected = lst[i].Selected.Toggle(m) }

	switch selection.(type) {
		// rows...
		case []Row:
			s := selection.([]Row)
			for i, _ := range s {
				toggle(s, i) }
		// indexes...
		case []int:
			s := selection.([]int)
			for _, i := range s {
				toggle(this.Lines, i) }
		// strings...
		case []string:
			s := selection.([]string)
			var i = 0
			for _, line := range s{
				for i < len(this.Lines) {
					if line == this.Lines[i].Text {
						toggle(this.Lines, i) }
					i++ }
				// loop over .Lines in case we've got the selection out of 
				// order...
				if i >= len(this.Lines) - 1 {
					i = 0 } } }
	return this }
func (this *LinesBuffer) SetSelection(selection any, mode ...Toggle) *LinesBuffer {
	this.SelectNone()
	var m Toggle
	if len(mode) == 0 {
		m = mode[0] }
	return this.Select(selection, m) }
func (this *LinesBuffer) SelectToggle(selection []any) *LinesBuffer {
	this.Select(this.Lines, Next)
	return this }
func (this *LinesBuffer) SelectAll() *LinesBuffer {
	this.Select(this.Lines, On)
	return this }
func (this *LinesBuffer) SelectNone() *LinesBuffer {
	this.Select(this.Lines, Off)
	return this }
func (this *LinesBuffer) ActiveRows() []Row {
	sel := this.SelectedRows()
	if len(sel) == 0 &&
			len(this.Lines) > 0 {
		sel = []Row{ this.Lines[this.Index] } }
	return sel }
func (this *LinesBuffer) Active() []string {
	sel := this.Selected()
	if len(sel) == 0 &&
			len(this.Lines) > 0 {
		sel = []string{ this.Current() } }
	return sel }




// vim:set ts=4 sw=4 :
