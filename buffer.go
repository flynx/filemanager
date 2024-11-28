
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

	// normalize inputs...
	for i, in := range strs {
		switch in.(type) {
			// readers make things non-deterministic...
			case io.Reader:
			case []byte:
				lst := strings.Split(string(in.([]byte)), "\n")
				strs[i] = lst
			case string:
				lst := strings.Split(in.(string), "\n")
				strs[i] = lst
			case []string:
			// convert any to string...
			default:
				strs[i] = fmt.Sprint(in) } }

	// append...
	n := 0
	place := func(s string) {
		this.__writing.Lock()
		defer this.__writing.Unlock()
		defer this.Changed.Trigger()
		n++
		this.Lines = append(this.Lines, 
			Row{
				Text: s,
				// XXX IGNORE_EMPTY...
				Populated: true,
				//Populated: len(s) != 0,
			}) }
	for _, in := range strs {
		switch in.(type) {
			case io.Reader:
				scanner := bufio.NewScanner(in.(io.Reader))
				for scanner.Scan() {
					place(scanner.Text()) }
			case []string:
				for _, in := range in.([]string) {
					place(in) } 
			case string:
				place(in.(string)) } }

	return n }
// XXX do we need .Write(..)
// XXX HACK: no error handling...
func (this *LinesBuffer) Write(b []byte) (int, error) {
	this.Clear()
	this.Append(string(b))
	return len(b), nil }


const (
	IterVisible = 0
	IterUnpopulated = 1 << iota
	IterBlank
) 
// XXX should this be live or clone the list???
//		...since we can modify the row.Text, doing a slice would require 
//		a deep copy...
func (this *LinesBuffer) Iter(modes... int) (func(func(*Row) bool)) {
	mode := IterVisible
	for _, m := range modes {
		mode += m }
	populated := mode & IterUnpopulated != 0
	blank := mode & IterBlank != 0
	return func(yield func(*Row) bool) {
		for i, _ := range this.Lines {
			row := &this.Lines[i]
			// skip stuff...
			if (!populated && 
						!row.Populated) || 
					(!blank && 
						len(row.Text) == 0) {
				continue }
			if !yield(row) {
				return } } } }

//
//	.Range()
//	.Range(<from>)
//	.Range(<from>, <to>)
//	.Range(<from>, <to>, <modes>...)
//		-> <iter>
//
// XXX add support for negative <to> to count from the back... (???)
//		can this be done in a live manner??
func (this *LinesBuffer) Range(args... int) (func(func(*Row) bool)) {
	from := 0
	to := -1
	modes := []int{}
	if len(args) > 0 {
		from = args[0] }
	if len(args) > 1 {
		to = args[1] }
	if len(args) > 2 {
		modes = args[2:] }

	return func(yield func(*Row) bool) {
		i := -1
		for row := range this.Iter(modes...) {
			i++
			// skip head...
			if i < from {
				continue }
			// drop tail...
			if to >= 0 && 
					i >= to {
				return }
			if !yield(row) {
				return } } } }


// XXX this is quite generic -- move to a better module...
func IterStepper[T any](iter func(func(T)bool)) (<-chan T) {
	c := make(chan T)
	go func(){
		iter(func(e T)bool{
			c <- e
			return true })
		close(c) }()
	return c }


// XXX is this a good idea???
func (this *LinesBuffer) At(index int) *Row {
	return <-IterStepper(this.Range(index)) }


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
// XXX is there a way to detect skipped callbacks without storing "from" 
//		in a closure?
//		...this is needed to set .Populated = false on skipped sources
//		...it would also be nice if we could create a single callback for
//		the whole map session...
//		(see: cli.go: UI.MapCmd())
// XXX would be nice to:
//		- block read until output (in "clear" mode?)
//			...this is a deadlock on skip -- the next input will not trigger...
//		- position-free (ignore seen)
// XXX we could sync on transformer(..) return -- i.e. when it returned 
//		then the input line is cleared (a-la from++)
// XXX handle empty...
func (this *LinesBuffer) PositionalMap(transformer Transformer, mode ...string) *LinesBuffer {
	this.Transformers = append(this.Transformers, transformer)
	level := len(this.Transformers)

	i := 0
	to := 0
	//* XXX FROM avoid per-element state (from) in the callback...
	//		...we do not care about the order of calls to callback(..)...
	//		from is used to detect skipped items...
	//		...without this we can not detect if a callback was skipped 
	//		and thus can't hide the skipped input...
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
			//if to >= len(this.Lines) ||
			//		this.Lines[to].Transformed != -level {
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
	/*/
	callback := func(s string){
		this.__writing.Lock()
		defer this.__writing.Unlock() 
		defer this.Changed.Trigger() 

		// handle inserts/shifts done to the left of us -- by higher level transforms...
		for to < len(this.Lines) && 
				this.Lines[to].Transformed >= level {
			i++
			to++ }

		// handle inserts...
		if to >= len(this.Lines) ||
				this.Lines[to].Transformed != -level {
			this.Lines = slices.Insert(this.Lines, to, Row{
				Transformed: -level,
				Populated: false,
			}) 
			i++ }

		// update the row...
		this.Lines[to].Text = s
		this.Lines[to].Populated = true
		this.Lines[to].Transformed = level 
		to++ }
	//*/

	// restart...
	this.Cleared.On(
		func(){
			i = 0
			//* XXX FROM
			to = 0
			seen = -1 })
			/*/
			to = 0 })
			//*/

	// feed this.Lines to transformer(..)
	go func(){
		// check if transformer removed...
		// XXX is this a good idea???
		__t := reflect.ValueOf(transformer).Pointer()
		isRemoved := func() bool {
			return ! slices.ContainsFunc(
				this.Transformers, 
				func(t Transformer) bool {
					return reflect.ValueOf(t).Pointer() == __t }) }

		// transform (infinite loop)...
		for ; true; i++ {
			if isRemoved() {
				return }

			// handle trim/reset...
			if len(this.Lines) < i {
				i = len(this.Lines)-1
				//* XXX FROM
				to = i
				seen = i-1 }
				/*/
				to = i }
				//*/

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
			/*/ skip empty and unpopulated lines...
			// XXX IGNORE_EMPTY
			if !row.Populated ||
					len(row.Text) == 0 {
				continue }
			//*/

			// mark row as read but not yet transformed...
			row.Transformed = -level
			// clear items before transform...
			if len(mode) > 0 && 
					mode[0] == "clear" {
				row.Populated = false }
			// NOTE: if transformer(..) calls callback(..) multiple times 
			//		it will update i...
			//* XXX FROM
			transformer(row.Text, callback(i)) } }()
			/*/
			transformer(row.Text, callback) } }()
			//*/

	return this }

// XXX IGNORE_EMPTY might be a good idea to mark filtered out lines by returning ""...
// XXX SYNC_OUT do we need this???
// XXX SYNC_OUT deadlock on unbuffered channel -- needs revision, is this
//		a solution??
//		(see: make(...) inside)
// XXX generic map -- generic callback with no concept of position...
//		...can we make this an option/mode???
// XXX the problem with clear by default with async handlers is that we'll 
//		clean the whole list before we get to see any updates...
//		...is strictly separating filter and map the only way to fix this???
//			map can be block input till output is done + auto output on return
//			...should we sync on channel (out) or on transformer(..) return??? (XXX SYNC_OUT)
// XXX dis does not pass all the tests...
// XXX revise mode...
// XXX handle empty...
func (this *LinesBuffer) SimpleMap(transformer Transformer, mode ...string) *LinesBuffer {
	this.Transformers = append(this.Transformers, transformer)
	level := len(this.Transformers)

	// mode default...
	if len(mode) == 0 {
		mode = append(mode, "clear") }

	i := 0
	to := 0
	// XXX SYNC_OUT
	// XXX an unbuffered channel here will block things on .Append(..)...
	//		...is adding a buffer a solution or is this simply shifting 
	//		the problem?
	//out := make(chan bool, 8)
	callback := func(s string){
		this.__writing.Lock()
		defer this.__writing.Unlock() 
		defer this.Changed.Trigger() 

		// handle inserts/shifts done to the left of us -- by higher level transforms...
		for to < len(this.Lines) && 
				this.Lines[to].Transformed >= level {
			i++
			to++ }

		// handle inserts...
		if to >= len(this.Lines) ||
				this.Lines[to].Transformed != -level {
			this.Lines = slices.Insert(this.Lines, to, Row{
				Transformed: -level,
				Populated: false,
			}) 
			i++ }

		// update the row...
		this.Lines[to].Text = s
		this.Lines[to].Populated = true
		this.Lines[to].Transformed = level 
		to++ 
		// allow next input...
		// XXX SYNC_OUT
		//out <- true 
	}

	// restart...
	this.Cleared.On(
		func(){
			i = 0
			to = 0 })

	// feed this.Lines to transformer(..)
	go func(){
		// check if transformer removed...
		// XXX is this a good idea???
		__t := reflect.ValueOf(transformer).Pointer()
		isRemoved := func() bool {
			return ! slices.ContainsFunc(
				this.Transformers, 
				func(t Transformer) bool {
					return reflect.ValueOf(t).Pointer() == __t }) }

		//lines := IterStepper(this.Iter())

		// transform (infinite loop)...
		for ; true; i++ {
			if isRemoved() {
				return }

			// handle trim/reset...
			if len(this.Lines) < i {
				i = len(this.Lines)-1
				to = i }

			// wait till a new value is available...
			for i >= len(this.Lines) ||
					this.Lines[i].Transformed < level-1 {
				this.Changed.Wait() 
				if isRemoved() {
					return } }

			row := &this.Lines[i]
			//row := <-lines

			// skip shifted items...
			if row.Transformed >= level {
				continue }
			/*/ skip empty and unpopulated lines...
			// XXX IGNORE_EMPTY
			if !row.Populated ||
					len(row.Text) == 0 {
				continue }
			//*/

			// mark row as read but not yet transformed...
			row.Transformed = -level
			// clear items before transform...
			if len(mode) > 0 && 
					mode[0] == "clear" {
				row.Populated = false }
			// NOTE: if transformer(..) calls callback(..) multiple times 
			//		it will update i...
			transformer(row.Text, callback) 

			// wait for output...
			// XXX SYNC_OUT
			//<-out 
		} }()

	return this }


// Like .Map(..) but all Rows not processed yet are .Populated = false, 
// i.e. will not be returned by ..String()...
// XXX which map to use???
func (this *LinesBuffer) FilterMap(transformer Transformer, mode ...string) *LinesBuffer {
	return this.SimpleMap(transformer, "clear") }
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
	//return this.Lines[this.Index].Text }
	return this.At(this.Index).Text }
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
		//sel = []Row{ this.Lines[this.Index] } }
		sel = []Row{ *this.At(this.Index) } }
	return sel }
func (this *LinesBuffer) Active() []string {
	sel := this.Selected()
	if len(sel) == 0 &&
			len(this.Lines) > 0 {
		sel = []string{ this.Current() } }
	return sel }




// vim:set ts=4 sw=4 :
