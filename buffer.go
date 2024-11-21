
package main

import (
	"reflect"
	"fmt"
	//"log"
	"io"
	"strings"
	"slices"
	"bufio"
	"sync"
)


// Event...
//
// XXX should this be a separate module???
type EventHandler func()
type Event struct {
	__chan chan bool
	__triggering sync.Mutex

	__handlers []EventHandler
}
func (this *Event) Wait() *Event {
	this.__triggering.Lock()
	if this.__chan == nil {
		this.__chan = make(chan bool) }
	this.__triggering.Unlock()
	<-this.__chan
	return this }
func (this *Event) Trigger() *Event {
	this.__triggering.Lock()
	if this.__chan != nil {
		defer close(this.__chan) }
	this.__chan = make(chan bool)
	this.__triggering.Unlock()

	for _, handler := range this.__handlers {
		handler() }

	return this }
func (this *Event) On(handler EventHandler) *Event {
	this.__handlers = append(this.__handlers, handler)
	return this }
func (this *Event) Clear() *Event {
	this.__handlers = []EventHandler{}
	return this }

// XXX RENAME TriggerEvent -> events.Trigger, etc.
func TriggerEvent(evt *Event) *Event {
	return evt.Trigger() }
func OnEvent(evt *Event, handler EventHandler) *Event {
	return evt.On(handler) }



// Togglers...
//
// XXX should this be a separate module???
// XXX add a multi toggle...
type Toggler interface {
	Toggle(bool) bool
}
type Togglable interface {
	Next()
	On()
	Off()
}
type BoolTogglable interface {
	Togglable
	Toggle(Toggle)
}
type MultiTogglable interface {
	Togglable
	Prev()
}


type Toggle int
const (
	Next Toggle = iota
	On
	Off
)
func (this Toggle) Toggle(in bool) bool {
	if this == Next {
		return ! in
	} else if this == On {
		return true }
	return false }


type BoolToggle bool
func (this BoolToggle) Toggle(mode Toggle) BoolToggle {
	if mode == Next {
		return this.Next()
	} else if mode == On {
		return true }
	return false }
func (this BoolToggle) Next() BoolToggle {
	return ! this }
func (this BoolToggle) On() BoolToggle {
	return true }
func (this BoolToggle) Off() BoolToggle {
	return false }



// Row
//
type Row struct {
	Selected BoolToggle
	Transformed int
	Populated bool
	Text string
}


// XXX
type TransformerCallback func(string)
type Transformer func(string, TransformerCallback)


// LinesBuffer
//
type LinesBuffer struct {
	sync.Mutex
	Lines []Row
	Index int

	Transformers []Transformer

	__writing sync.Mutex
	__appending sync.Mutex

	// events...
	Changed Event
	Cleared Event

}


func (this *LinesBuffer) Len() int {
	l := 0
	for _, row := range this.Lines {
		if row.Populated {
			l++ } }
	return l }


// Editing...
//
func (this *LinesBuffer) Clear() *LinesBuffer {
	this.Lines = []Row{}
	this.Index = 0
	this.Cleared.Trigger()
	return this }

func (this *LinesBuffer) String() string {
	lines := []string{}
	//for i, line := range this.Lines {
	for _, line := range this.Lines {
		if ! line.Populated {
			continue }
		lines = append(lines, line.Text) }
	return strings.Join(lines, "\n") }

func (this *LinesBuffer) Trim() *LinesBuffer {
	this.__writing.Lock()
	defer this.__writing.Unlock()
	// XXX is this a good idea???
	this.__appending.Lock()
	defer this.__appending.Unlock()
	defer this.Changed.Trigger()

	to := 0
	for from := 0; from < len(this.Lines); from++ {
		populated := this.Lines[from].Populated
		// skip...
		if ! populated {
			continue }
		// shift...
		if populated && 
				to != from {
			this.Lines[to] = this.Lines[from]
			to++ 
			continue }
		// no change...
		to++ }
	// truncate...
	this.Lines = this.Lines[:to]
	return this }

func (this *LinesBuffer) Append(strs ...any) int {
	this.__appending.Lock()
	defer this.__appending.Unlock()
	//this.__writing.Lock()
	//defer this.__writing.Unlock()
	defer this.Changed.Trigger()

	i := len(this.Lines)
	// get the write position...
	for ; i > 0; i-- {
		if this.Lines[i-1].Populated {
			break } }

	l := 0
	// normalize inputs + count lines if possible...
	for i, in := range strs {
		switch in.(type) {
			// readers make things non-deterministic...
			case io.Reader:
				l = -1
				//defer this.__writing.Unlock() 
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
	/* XXX do we need this???
	// grow .Lines if needed...
	if l >= 0 {
		//this.Length += l 
		//if this.Length > len(this.Lines) {
		//	slices.Grow(this.Lines, this.Length - len(this.Lines)) }
		this.__writing.Unlock() }
	//*/

	// append...
	//
	place := func(i int, s string) bool {
		row := Row{
			Text: s,
			Populated: true,
		}
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

// Transforms / Map...
//
//	.Transform(transformer[, mode])
//
//	mode:
//		"clear"		- set each item to .Populated = false before transforming.
//

// NOTE: removing a transformer from .Transformers will stop it from 
//		running, it will exit after the next .Changed, to force this 
//		trigger the event manually (i.e. call .Changed.Trigger())
// NOTE: we do not care about callback(..) call order here -- sequencing 
//		callback(..) calls should be done by transformer(..)
func (this *LinesBuffer) Map(transformer Transformer, mode ...string) *LinesBuffer {
	this.Transformers = append(this.Transformers, transformer)
	level := len(this.Transformers)

	// XXX is this a good idea???
	__t := reflect.ValueOf(transformer).Pointer()
	isRemoved := func() bool {
		return ! slices.ContainsFunc(
			this.Transformers, 
			func(t Transformer) bool {
				return reflect.ValueOf(t).Pointer() == __t }) }

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
				for i := to+1; i <= from && i < len(this.Lines); i++ {
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
			if isRemoved() {
				return }

			// handle trim/reset...
			if len(this.Lines) < i {
				i = len(this.Lines)-1
				to = i
				seen = i-1 }

			// wait till a new value is available...
			for i >= len(this.Lines) ||
					this.Lines[i].Transformed < level-1 {
				this.Changed.Wait() 
				if isRemoved() {
					return } }


			row := &this.Lines[i]

			// skip shifted items...
			if row.Transformed >= level {
				continue }

			// mark row as read but not yet transformed...
			row.Transformed = -level
			// clear items before transform...
			if len(mode) > 0 && 
					mode[0] == "clear" {
				row.Populated = false }
			// NOTE: if transformer(..) calls callback(..) multiple times 
			//		it will update i...
			transformer(row.Text, callback(i)) } }()

	return this }
// Like .Map(..) but all Rows not processed yet are .Populated = false, 
// i.e. will not be returned by ..String()...
func (this *LinesBuffer) FMap(transformer Transformer, mode ...string) *LinesBuffer {
	return this.Map(transformer, "clear") }
func (this *LinesBuffer) ClearTransforms(t ...bool) *LinesBuffer {
	this.Transformers = []Transformer{}
	// force the transformers to self-remove...
	if len(t) > 0 && t[0] == true {
		this.Changed.Trigger() }
	return this }


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
