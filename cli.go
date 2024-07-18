/*
* XXX split this into two parts:
*		- tcell
*			- drawing
*			- event loop
*		- generic
*			- config
*			- cli
*			- key/click handlers
*		- actions (???)
*/
package main

import (
	//"fmt"
	"log"
	"os"
	"syscall"
	"reflect"
	"sync"
	"time"
	//"io"
	"bufio"
	"runtime"
	//"bytes"
	"strings"
	"unicode"
	"strconv"
	"slices"
	//"regexp"

	"github.com/gdamore/tcell/v2"
	//"github.com/jessevdk/go-flags"

	//"github.com/mkideal/cli"
)


// Tcell-specific helpers...
//
func Style2TcellStyle(style Style, base_style ...tcell.Style) tcell.Style {
	base := tcell.StyleDefault
	if len(base_style) != 0 {
		base = base_style[0] } 
	// set flags...
	colors := []string{}
	for _, s := range style {
		switch s {
			case "blink":
				base = base.Blink(true)
			case "bold":
				base = base.Bold(true)
			case "dim":
				base = base.Dim(true)
			case "italic":
				base = base.Italic(true)
			case "normal":
				base = base.Normal()
			case "reverse":
				base = base.Reverse(true)
			case "strike-through":
				base = base.StrikeThrough(true)
			case "underline":
				base = base.Underline(true)
			default:
				// urls...
				if string(s[:len("url")]) == "url" {
					p := strings.SplitN(s, ":", 2)
					url := ""
					if len(p) > 1 {
						url = p[1] }
					base = base.Url(url)
				// colors...
				} else {
					colors = append(colors, s) } } }
	// set the colors...
	if len(colors) > 0 && 
			colors[0] != "as-is" {
		base = base.Foreground(
			tcell.GetColor(colors[0])) }
	if len(colors) > 1 &&
			colors[1] != "as-is" {
		base = base.Background(
			tcell.GetColor(colors[1])) }
	return base }
func TcellEvent2Keys(evt tcell.EventKey) []string {
	mods := []string{}
	shifted := false

	var key, Key string

	mod, k, r := evt.Modifiers(), evt.Key(), evt.Rune()

	// handle key and shift state...
	if k == tcell.KeyRune {
		if unicode.IsUpper(r) {
			shifted = true
			Key = string(r)
			mods = append(mods, "shift") }
		key = string(unicode.ToLower(r))
	// special keys...
	} else if k > tcell.KeyRune || k <= tcell.KeyDEL {
		key = evt.Name()
	// ascii...
	} else {
		if unicode.IsUpper(rune(k)) {
			shifted = true 
			Key = string(rune(k))
			mods = append(mods, "shift") } 
		key = strings.ToLower(string(rune(k))) } 

	// split out mods and normalize...
	key_mods := strings.Split(key, "+")
	key = key_mods[len(key_mods)-1]
	if k := []rune(key) ; len(k) == 1 && unicode.IsUpper(k[0]) {
		key = strings.ToLower(key) }
	key_mods = key_mods[:len(key_mods)-1]

	// basic translation...
	if key == " " {
		key = "Space" }

	if slices.Contains(key_mods, "Ctrl") || 
			mod & tcell.ModCtrl != 0 {
		mods = append(mods, "ctrl") }
	if slices.Contains(key_mods, "Alt") || 
			mod & tcell.ModAlt != 0 {
		mods = append(mods, "alt") }
	if slices.Contains(key_mods, "Meta") || 
			mod & tcell.ModMeta != 0 {
		mods = append(mods, "meta") }
	if !shifted && mod & tcell.ModShift != 0 {
		mods = append(mods, "shift") }

	return key2keys(mods, key, Key) }



// Keyboard...
//
type KeyAliases = map[string][]string
var KEY_ALIASES = KeyAliases {
	/* XXX is this correct??
	//		...for some reason ctrl+Backspace2 is triggered as Backspace...
	"Backspace1": []string{ 
		"Backspace", 
		"Bkspace",
	},
	"Backspace2": []string{ 
		"Backspace",
		"Bkspace",
	},
	//*/
	"PgUp": []string{ 
		"PageUp", 
	},
	"PgDn": []string{ 
		"PageDown",
		"PgDown",
	},
	"Space": []string{ 
		" ",
	},
	"MouseLeft": []string{ 
		"Click",
		"LClick",
		"LeftClick",
	},
	"MouseRight": []string{ 
		"RClick",
		"RightClick",
	},
	"MouseMiddle": []string{ 
		"MClick",
	},
}
// XXX load this from config...
type Keybindings map[string]string
var KEYBINDINGS = Keybindings {
	// aliases...
	"Select": "",
	"Reject": "Exit",

	// keys...
	//"Esc": "Reject",
	"Esc": "Down",
	"q": "Reject",
	"ctrl+z": "Stop",

	"Up": "Up",
	"Down": "Down",

	"WheelUp": "ScrollUp",
	"WheelDown": "ScrollDown",

	"PgUp": "PageUp",
	"PgDn": "PageDown",
	"Home": "Top",
	"End": "Bottom",

	"Enter": "Select",
	// XXX should we also have a "Click" event
	"ClickSelected": "Select",

	// XXX testing...
	//"x": "! echo \"$SELECTION\" > selection",
	//"a": "A=! A=${A:-1} echo $(( A + 1 ))",
	//"w": "! echo $A >> sum.log",

	// Mouse...
	"Click": "Focus",

	// Selection...
	// XXX shift is not detected on most terminals...
	//"shift": `
	//	SelectStart`,
	"ctrl+Click": `
		Focus
		SelectToggle`,
	"Insert": `
		SelectToggle
		Down`,
	"Space": `
		SelectToggle
		Down`,
	// XXX for some reason shift+click is not even handled...
	"shift+Click": `
		SelectStart
		Focus
		SelectEndCurrent`,
	"shift+Up": `
		SelectStart
		Up
		SelectEnd`,
	"shift+Down": `
		SelectStart
		Down
		SelectEnd`,
	"shift+PgUp": `
		SelectStart
		PageUp
		SelectEndCurrent`,
	"shift+PgDn": `
		SelectStart
		PageDown
		SelectEndCurrent`,
	"shift+Home": `
		SelectStart
		Top
		SelectEndCurrent`,
	"shift+End": `
		SelectStart
		Bottom
		SelectEndCurrent`,
	"ctrl+a": "SelectAll",
	// XXX ctrl-i is Tab -- can we destinguish the two in the terminal???
	"ctrl+i": "SelectInverse",
	"^": "SelectInverse",
	"ctrl+d": "SelectNone",

	"ctrl+r": "Update",
	"ctrl+l": "Refresh",
}

func key2keys(mods []string, key string, rest ...string) []string {
	key_seq := []string{}
	Key := ""
	if len(rest) > 0 {
		Key = rest[0] }

	// XXX STUB -- still need 3 and 4 mod combinations for completeness...
	//		...generate combinations + sort by length...
	for i := 0; i < len(mods); i++ {
		for j := i+1; j < len(mods); j++ {
			key_seq = append(key_seq, mods[i] +"+"+ mods[j] +"+"+ key) } }
	for _, m := range mods {
		key_seq = append(key_seq, m +"+"+ key) }
	// uppercase letter...
	if Key != "" {
		key_seq = append(key_seq, Key) }
	key_seq = append(key_seq, key)

	return key_seq }



// Result...
//
type Result int
const (
	// Normal action return value.
	OK Result = -1 + iota

	// Returning this from an action will quit lines (exit code 0)
	Exit 

	// Returning this will quite lines with error (exit code 1)
	Fail

	// Action missing and can not be called -- test next or fail
	Missing
)
// Convert from Result type to propper exit code.
func toExitCode(r Result) int {
	if r == Fail {
		return int(Fail) }
	return 0 }



// Actions...
//
type Actions struct {
	// XXX this needs to be more generic...
	*TcellDrawer

	last string

	// can be:
	//	"select"
	//	"deselect"
	//	""			- toggle
	SelectMotion string
	SelectMotionStart int
}

func NewActions(d *TcellDrawer) *Actions {
	return &Actions{
		TcellDrawer: d,
	} }

// base action...
// update the .last attr with the action name...
func (this *Actions) Action() Result {
	pc, _, _, ok := runtime.Caller(1)
	this.last = ""
	if ok {
		path := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		this.last = path[len(path)-1] }
	return OK }

// Debug helper...
func (this *Actions) LOG() Result {
	log.Println("ACTION: LOG")
	return OK }

// General...
func (this *Actions) Focus() Result {
	// second click on same row...
	if this.MouseRow == this.Lines.Index {
		res := this.HandleKey("ClickSelected") 
		if res == Missing {
			res = OK }
		if res != OK {
			return res } }
	// select row...
	this.Lines.Index = this.MouseRow
	return OK }

// Vertical navigation...
func (this *Actions) Up() Result {
	this.Action()
	if this.Lines.Index > 0 && 
			// at scroll threshold (top)...
			(this.Lines.Index > this.ScrollThreshold ||
				this.Lines.RowOffset == 0) {
		this.Lines.Index-- 
	// scroll the buffer...
	} else {
		this.ScrollUp() }
	return OK }
func (this *Actions) Down() Result {
	this.Action()
	rows := this.Lines.Rows()
	// within the text buffer...
	if this.Lines.Index + this.Lines.RowOffset < len(this.Lines.Lines)-1 && 
			// within screen...
			this.Lines.Index < rows-1 && 
			// buffer smaller than screen...
			(rows >= len(this.Lines.Lines) ||
				// screen at end of buffer...
				this.Lines.RowOffset + rows == len(this.Lines.Lines) ||
				// at scroll threshold (bottom)...
				this.Lines.Index < rows - this.ScrollThreshold - 1) {
		this.Lines.Index++ 
	// scroll the buffer...
	} else {
		this.ScrollDown() }
	return OK }

// XXX should these track this.Lines.Index relative to screen (current) or 
//		relative to content???
func (this *Actions) ScrollUp() Result {
	this.Action()
	if this.Lines.RowOffset > 0 {
		this.Lines.RowOffset-- }
	return OK }
func (this *Actions) ScrollDown() Result {
	this.Action()
	rows := this.Lines.Rows()
	if this.Lines.RowOffset + rows < len(this.Lines.Lines) {
		this.Lines.RowOffset++ } 
	return OK }

func (this *Actions) PageUp() Result {
	this.Action()
	if this.Lines.RowOffset > 0 {
		rows := this.Lines.Rows()
		this.Lines.RowOffset -= rows 
		if this.Lines.RowOffset < 0 {
			this.Top() } 
	} else if this.Lines.RowOffset == 0 {
		this.Top() } 
	return OK }
func (this *Actions) PageDown() Result {
	this.Action()
	rows := this.Lines.Rows()
	if len(this.Lines.Lines) < rows {
		this.Lines.Index = len(this.Lines.Lines) - 1
		return OK }
	offset := len(this.Lines.Lines) - rows
	if this.Lines.RowOffset < offset {
		this.Lines.RowOffset += rows 
		if this.Lines.RowOffset > offset {
			this.Bottom() } 
	} else if this.Lines.RowOffset == offset {
		this.Bottom() } 
	return OK }

func (this *Actions) Top() Result {
	this.Action()
	if this.Lines.RowOffset == 0 {
		this.Lines.Index = 0 
	} else {
		this.Lines.RowOffset = 0 }
	return OK }
func (this *Actions) Bottom() Result {
	this.Action()
	rows := this.Lines.Rows()
	if len(this.Lines.Lines) < rows {
		this.Lines.Index = len(this.Lines.Lines) - 1
		return OK }
	offset := len(this.Lines.Lines) - rows 
	if this.Lines.RowOffset == offset {
		this.Lines.Index = rows - 1
	} else {
		this.Lines.RowOffset = len(this.Lines.Lines) - rows }
	return OK }
//*/

/*// XXX Horizontal navigation...
func (this *Actions) Left() Result {
	this.Action()
	// XXX
	return OK }
func (this *Actions) Right() Result {
	this.Action()
	// XXX
	return OK }

func (this *Actions) ScrollLeft() Result {
	this.Action()
	// XXX
	return OK }
func (this *Actions) ScrollRight() Result {
	this.Action()
	// XXX
	return OK }

func (this *Actions) LeftEdge() Result {
	this.Action()
	// XXX
	return OK }
func (this *Actions) RightEdge() Result {
	this.Action()
	// XXX
	return OK }
//*/

// Selection...
// NOTE: the selection is expected to mostly be in order.
// XXX would be nice to be able to match only left/right side of span...
//		...not sure how to configure this...
func (this *Actions) Select(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{this.Lines.Index + this.Lines.RowOffset} }
	for _, i := range rows {
		this.Lines.Lines[i].Selected = true }
	return OK }
func (this *Actions) Deselect(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{this.Lines.Index + this.Lines.RowOffset} }
	for _, i := range rows {
		this.Lines.Lines[i].Selected = false }
	return OK }
// XXX us this.Lines.SelectToggle(..)
func (this *Actions) SelectToggle(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{this.Lines.Index + this.Lines.RowOffset} }
	for _, i := range rows {
		if this.Lines.Lines[i].Selected {
			this.Lines.Lines[i].Selected = false 
		} else {
			this.Lines.Lines[i].Selected = true } }
	return OK }
func (this *Actions) SelectAll() Result {
	this.Action()
	this.Lines.SelectAll()
	return OK }
func (this *Actions) SelectNone() Result {
	this.Action()
	this.Lines.SelectNone()
	return OK }
func (this *Actions) SelectInverse() Result {
	this.Action()
	rows := []int{}
	for i := 0 ; i < len(this.Lines.Lines) ; i++ {
		rows = append(rows, i) }
	return this.SelectToggle(rows...) }

// BUG/FEATURE: after releasing shift and then pressing it again with 
//		motion a new selection session is not started rather the old one 
//		is continud...
//		...this is not a bug, we cant do anything about it unless we can 
//		detect shift key press/release...
func (this *Actions) SelectStart() Result {
	if this.last != "SelectEnd" {
		log.Println("NEW SELECTION")
		this.SelectMotion = "select"
		if this.Lines.Lines[this.Lines.Index+this.Lines.RowOffset].Selected {
			this.SelectMotion = "deselect" } }
	log.Println("SELECTION", this.SelectMotion)
	this.Action()
	this.SelectMotionStart = this.Lines.Index + this.Lines.RowOffset
	return OK }
// XXX need to handle shift release...
func (this *Actions) SelectEnd(rows ...int) Result {
	this.Action()
	var start, end int
	if len(rows) >= 2 {
		start, end = rows[0], rows[1] 
	} else if len(rows) == 1 {
		start, end = this.SelectMotionStart, rows[0]
	} else {
		start = this.SelectMotionStart
		end = this.Lines.Index + this.Lines.RowOffset 
		/* XXX
		// toggle selection mode when on first/last row...
		// XXX HACK? this should be done on shift release...
		if start == end && 
				(end == 0 || 
				end == len(this.Lines.Lines)-1) {
			if this.Lines.Lines[end].Selected && 
					this.SelectMotion == "select" {
				this.SelectMotion = "deselect"
			} else if ! this.Lines.Lines[end].Selected && 
					this.SelectMotion == "deselect" {
				this.SelectMotion = "select" } }
		//*/
		// do not select the current item unless we start on it...
		if end < start {
			end++
		} else if end > start {
			end-- } }
	// normalize direction...
	if this.SelectMotionStart > end {
		start, end = end, start }
	lines := []int{}
	for i := start ; i <= end; i++ {
		lines = append(lines, i) }
	if this.SelectMotion == "select" {
		this.Select(lines...)
	} else if this.SelectMotion == "deselect" {
		this.Deselect(lines...)
	} else {
		this.SelectToggle(lines...) }
	this.Action()
	return OK }
func (this *Actions) SelectEndCurrent() Result {
	return this.SelectEnd(this.Lines.Index + this.Lines.RowOffset) }

// Utility...
/* XXX
// XXX revise behaviour of reupdates on pipe...
func (this *Actions) Update() Result {
	selection := this.Lines.Selected()
	res := OK
	// file...
	if INPUT_FILE != "" {
		file, err := os.Open(INPUT_FILE)
		if err != nil {
			fmt.Println(err)
			return Fail }
		defer file.Close()
		this.Lines.Write(file) 
	// command...
	} else if LIST_CMD != "" {
		res = callAction("<"+ LIST_CMD)
	// pipe...
	// XXX how should this behave on re-update???
	//		...should we replace, append or simply redraw cache???
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalf("%+v", err) }
		if stat.Mode() & os.ModeNamedPipe != 0 {
			// XXX do we need to close this??
			//defer os.Stdin.Close()
			this.Lines.Write(os.Stdin) } }
	this.Lines.Select(selection)
	if FOCUS_CMD != "" {
		// XXX generate FOCUS
	}
	//this.Refresh()
	return res }
//*/
func (this *Actions) Refresh() Result {
	this.TcellDrawer.Refresh()
	return OK }

// XXX will this stop goroutines?? (TEST)
// XXX will this work on windows / windows version / disable on windows???
//		...how do we deal with things like cygwin/MinGW/..???
func (this *Actions) Stop() Result {
	screen := this.TcellDrawer.Screen
	_, ok := screen.Tty()
	if ! ok {
		return OK }
	// XXX can we go around all of this and simple pass ctrl-z to parent???
	screen.Suspend()
	pid := syscall.Getppid()
	// ask parent to detach us from io...
	err := syscall.Kill(pid, syscall.SIGSTOP)
	if err != nil {
		log.Panic(err) }
	// stop...
	err = syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	if err != nil {
		log.Panic(err) }
	time.Sleep(time.Millisecond*50)
	screen.Resume()
	return OK }
func (this *Actions) Fail() Result {
	return Fail }
func (this *Actions) Exit() Result {
	return Exit }



// Drawer...

var REFRESH_INTERVAL = time.Millisecond * 15

//
// XXX renmae...
type TcellDrawer struct {
	tcell.Screen

	Lines *Lines

	Actions *Actions

	// Keyboard..
	//
	KeyAliases KeyAliases
	Keybindings Keybindings

	// Geometry
	//
	// Format:
	//		"auto" | "50%" | "20"
	Width string
	Height string

	// Format:
	//		<value> ::= {<left>, <top>}	
	//		<left> ::= "left" | "center" | "right" | "42"
	//		<top> ::= "top" | "center" | "bottom" | "42"
	Align []string


	// state...
	MouseRow int
	MouseCol int
	ScrollThreshold int
	// Format:
	//		"" | "active"
	EmptySpace string


	Transformer *PipedCmd

	// caches...
	// NOTE: in normal use-cases the stuff cached here is static and 
	//		there should never be any leakage, if there is then something 
	//		odd is going on.
	__style_cache map[string]tcell.Style
	__float_cache map[string]float64
	//__int_cache map[string]int

	RefreshInterval time.Duration
	__refresh_blocked sync.Mutex
	__refresh_pending sync.Mutex
}

func (this *TcellDrawer) updateGeometry() *TcellDrawer {
	var err error
	W, H := this.Screen.Size()
	Width, Height := this.Width, this.Height
	Align := this.Align
	if len(Align) == 0 {
		Align = []string{"left", "top"} }

	// XXX should this be more generic???
	// XXX revise the error case...
	cachedFloat := func(str string) float64 {
		if this.__float_cache == nil {
			this.__float_cache = map[string]float64{} }
		v, ok := this.__float_cache[str]
		if ! ok {
			var err error
			// handle "%"...
			if str[len(str)-1] == '%' {
				str = string(str[0:len(str)-1]) }
			v, err = strconv.ParseFloat(str, 32)
			if err != nil {
				log.Println(err) } 
			this.__float_cache[str] = v }
		return v }

	// Width...
	if Width == "auto" || 
			Width == "" {
		this.Lines.Width = W
	} else if Width[len(Width)-1] == '%' {
		this.Lines.Width = int(float64(W) * (cachedFloat(Width) / 100))
	} else {
		// XXX revise the error case + cache???
		this.Lines.Width, err = strconv.Atoi(Width)
		if err != nil {
			log.Println("Error parsing width", Width) } }
	// Height...
	if Height == "auto" || 
			Height == "" {
		this.Lines.Height = H
	} else if Height[len(Height)-1] == '%' {
		this.Lines.Height = int(float64(H) * (cachedFloat(Height) / 100))
	} else {
		// XXX revise the error case + cache???
		this.Lines.Height, err = strconv.Atoi(Height)
		if err != nil {
			log.Println("Error parsing height", Height) } }

	// Left (value)
	left_set := false
	if slices.Contains(Align, "left") {
		left_set = false
		this.Lines.Left = 0
	} else if slices.Contains(Align, "right") {
		left_set = false
		this.Lines.Left = W - this.Lines.Width
	} else if len(Align) > 0 &&
			Align[0] != "center" {
		left_set = false
		// XXX revise the error case + cache???
		this.Lines.Left, err = strconv.Atoi(Align[0])
		if err != nil {
			log.Println("Error parsing left", Align[0]) } }
	// Top (value)
	top_set := false
	if slices.Contains(Align, "top") {
		top_set = false
		this.Lines.Top = 0
	} else if slices.Contains(Align, "bottom") {
		top_set = false
		this.Lines.Top = H - this.Lines.Height
	} else if len(Align) > 1 && 
			Align[1] != "center" {
		top_set = false
		// XXX revise the error case + cache???
		this.Lines.Top, err = strconv.Atoi(Align[1]) 
		if err != nil {
			log.Println("Error parsing top", Align[1]) } }
	// Left (center)
	if ! left_set {
		if top_set && 
				slices.Contains(Align, "center") || 
				(len(Align) > 0 && 
					Align[0] == "center") {
			this.Lines.Left = int(float64(W - this.Lines.Width) / 2) } }
	// Top (center)
	if ! top_set {
		if top_set && 
				slices.Contains(Align, "center") || 
				(len(Align) > 0 && 
					Align[0] == "center") {
			this.Lines.Top = int(float64(H - this.Lines.Height) / 2) } }
	return this }
// keep the selection in the same spot...
func (this *TcellDrawer) handleScrollLimits() *TcellDrawer {
	delta := 0

	rows := this.Lines.Rows()
	top_threshold := this.ScrollThreshold
	bottom_threshold := rows - this.ScrollThreshold - 1 
	if rows < this.ScrollThreshold + this.ScrollThreshold {
		top_threshold = rows / 2
		bottom_threshold = rows - top_threshold }
	
	// buffer smaller than screen -- keep at top...
	if rows > len(this.Lines.Lines) {
		this.Lines.RowOffset = 0
		// XXX this is odd -- see above line...
		this.Lines.Index -= this.Lines.RowOffset
		return this }

	// keep from scrolling past the bottom of the screen...
	if this.Lines.RowOffset + rows > len(this.Lines.Lines) {
		delta = this.Lines.RowOffset - (len(this.Lines.Lines) - rows)
	// scroll to top threshold...
	} else if this.Lines.Index < top_threshold && 
			this.Lines.RowOffset > 0 {
		delta = top_threshold - this.Lines.Index
		if delta > this.Lines.RowOffset {
			delta = this.Lines.RowOffset }
	// keep current row on screen...
	} else if this.Lines.Index > bottom_threshold && 
			this.Lines.Index > top_threshold {
		delta = bottom_threshold - this.Lines.Index
		// saturate delta...
		if delta < (this.Lines.RowOffset + rows) - len(this.Lines.Lines) {
			delta = (this.Lines.RowOffset + rows) - len(this.Lines.Lines) } } 

	// do the update...
	if delta != 0 {
		this.Lines.RowOffset -= delta 
		this.Lines.Index += delta }

	return this }

func (this *TcellDrawer) ResetCache() *TcellDrawer {
	this.__style_cache = nil
	this.__float_cache = nil
	//this.__int_cache = nil
	return this }

// Extends Style2TcellStyle(..) by adding cache...
//
// XXX do we need this public???
// XXX URLS are supported but not usable yet as there is no way to set 
//		the url...
//		use: "url:<url>"
// XXX would be nice to be able to use "foreground" and "background" 
//		colors in a predictable manner -- currently they reference curent 
//		colors
//		...i.e. {"yellow", "foreground"} will set both colors to yellow...
func (this *TcellDrawer) style2TcellStyle(style_name string, style Style) tcell.Style {
	// cache...
	if this.__style_cache == nil {
		this.__style_cache = map[string]tcell.Style{} }
	s, ok := this.__style_cache[style_name]
	if ok {
		return s }
	cache := func(s tcell.Style) tcell.Style {
		this.__style_cache[style_name] = s 
		return s }

	// base style (cached manually)...
	base, ok := this.__style_cache["default"]
	if ! ok {
		_, s := this.Lines.GetStyle("default")
		base = Style2TcellStyle(s) 
		this.__style_cache["default"] = base } 

	return cache(
		Style2TcellStyle(style, base)) }
func (this *TcellDrawer) drawCells(col, row int, str string, style_name string, style Style) {
	if style_name == "EOL" {
		return }
	s := this.style2TcellStyle(style_name, style)
	for i, r := range []rune(str) {
		this.SetContent(col+i, row, r, nil, s) } }
func (this *TcellDrawer) Fill() *TcellDrawer {
	_, s := this.Lines.GetStyle("background")
	this.Screen.Fill(' ', this.style2TcellStyle("background", s))
	return this }
func (this *TcellDrawer) Draw() *TcellDrawer {
	this.
		handleScrollLimits().
		// XXX do this separately...
		Fill().
		Lines.Draw()
	return this }
// NOTE: this will not refresh faster than once per .RefreshInterval
func (this *TcellDrawer) Refresh() *TcellDrawer {
	// refresh now...
	if this.__refresh_blocked.TryLock() {
		this.
			updateGeometry().
			Draw().
			Screen.Sync()
		this.Screen.Show()
		go func(){
			t := this.RefreshInterval
			if t == 0 {
				t = REFRESH_INTERVAL }
			time.Sleep(t)
			this.__refresh_blocked.Unlock() 
			if ! this.__refresh_pending.TryLock() {
				this.__refresh_pending.Unlock() 
				this.Refresh() } }() 
	// schedule a refresh...
	} else {
		this.__refresh_pending.TryLock() }
	return this }

// XXX not done yet...
func (this *TcellDrawer) HandleAction(actions string) Result {
	// XXX make split here a bit more cleaver:
	//		- support ";"
	//		- support quoting of separators, i.e. ".. \\\n .." and ".. \; .." -- DONE
	//		- ignore string literal content...
	parts := strings.Split(actions, "\n") 
	// merge back escaped "\n"'s...
	for i := 0 ; i < len(parts) ; i++ {
		part := parts[i]
		trimmed_part := strings.TrimSpace(part)
		if trimmed_part != "" && 
				trimmed_part[len(trimmed_part)-1] == '\\' {
			parts[i] += "\n"+ parts[i+1]
			if i < len(parts) + 1 {
				parts = append(parts[:i+1], parts[i+2:]...) } 
			i-- } }
	for _, action := range parts {
		action = strings.TrimSpace(action)
		if len(action) == 0 {
			continue }

		// shell commands:
		//		<NAME>=<CMD>	- command stdout read into env variable
		//		@ <CMD>			- simple/interactive command
		//		   					NOTE: this uses os.Stdout...
		//		! <CMD>			- stdout treated as env variables, one per line
		//		< <CMD>			- stdout read into buffer
		//		> <CMD>			- stdout printed to lines stdout
		//		| <CMD>			- current line passed to stdin
		//		XXX & <CMD>		- async command (XXX not implemented...)
		// NOTE: commands can be combined.
		// NOTE: if prefix combined with <NAME>=<CMD> it must come after "="
		/* XXX
		prefixes := "@!<>|"
		prefix := []rune{}
		code := action
		name := ""
		// <NAME>=<CMD>...
		if isVarCommand.Match([]byte(action)) {
			parts := regexp.MustCompile("=").Split(action, 2)
			name, action = parts[0], parts[1] 
			// <NAME>= -> remove from env...
			action = strings.TrimSpace(action)
			if name != "" && 
					(action == "" ||
						// prevent "<NAME>=<PREFIX>" with empty command 
						// from messing going through the whole call dance...
						(len(action) == 1 &&
							strings.ContainsRune(prefixes, rune(action[0])))){
				delete(ENV, name) 
				continue } }
		// <PREFIX><CMD>...
		for strings.ContainsRune(prefixes, rune(code[0])) {
			prefix = append(prefix, rune(code[0]))
			code = strings.TrimSpace(string(code[1:])) }
		if name != "" || 
				len(prefix) > 0 {
			var stdin bytes.Buffer
			if slices.Contains(prefix, '|') {
				stdin.Write([]byte(this.Lines.Lines[this.Lines.Index].text)) }

			// call the command...
			var err error
			var stdout *io.ReadCloser//bytes.Buffer
			lines := []string{}
			if slices.Contains(prefix, '@') {
				// XXX make this async...
				err = callAtCommand(code, stdin)
			} else {
				cmd := goCallCommand(code, &stdin)
				err = cmd.Error
				stdout = cmd.Stdout }
			if err != nil {
				log.Println("Error:", err)
				return Fail }

			// list output...
			// XXX keep selection and current item and screen position 
			//		relative to current..
			if slices.Contains(prefix, '<') {
				scanner := bufio.NewScanner(*stdout)
				// XXX should this be a goroutine/async???
				// XXX do the spinner....
				//this.Lines = LinesBuffer{}
				this.Lines.Lock()
				this.Lines.Clear()
				for scanner.Scan() {
					//// update screen as soon as we reach selection and 
					//// just after we fill the screen...
					//if len(this.Lines.Lines) == this.Lines.Index || 
					//		len(this.Lines.Lines) == CURRENT_ROW + this.Lines.RowOffset {
					//	this.Lines.Unlock() }
					line := scanner.Text()
					lines = append(lines, line)
					this.Lines.Push(line) } 
				if len(this.Lines.Lines) == 0 {
					this.Lines.Push("") } 
				this.Lines.Unlock()
				if FOCUS_CMD != "" {
					// XXX set focus...
				}
			// collect output data...
			} else if(stdout != nil) {
				scanner := bufio.NewScanner(*stdout)
				for scanner.Scan() {
					lines = append(lines, scanner.Text()) } }
			// output to stdout...
			if slices.Contains(prefix, '>') {
				STDOUT += strings.Join(lines, "\n") + "\n" }
			// output to env...
			if slices.Contains(prefix, '!') {
				for _, str := range lines {
					if strings.TrimSpace(str) == "" ||
							! isVarCommand.Match([]byte(str)) {
						continue }
					res := strings.SplitN(str, "=", 1)
					if len(res) != 2 {
						continue }
					ENV[strings.TrimSpace(res[0])] = strings.TrimSpace(res[1]) } }

			// handle env...
			if name != "" {
				if name == "STDOUT" {
					if ! slices.Contains(prefix, '>') {
						STDOUT += strings.Join(lines, "\n") + "\n" }
				} else {
					ENV[name] = strings.Join(lines, "\n") } }

		// ACTION...
		} else {
		//*/
			method := reflect.ValueOf(this.Actions).MethodByName(action)
			// test if action exists....
			if ! method.IsValid() {
				log.Println("Error: Unknown action:", action) 
				continue }
			res := method.Call([]reflect.Value{}) 
			// exit if action returns false...
			value, ok := res[0].Interface().(Result)
			if ! ok {
				// XXX
			}
			if value != OK {
				return value } } //}
	return OK }
func (this *TcellDrawer) HandleKey(key string) Result {
	keybindings := this.Keybindings
	if keybindings == nil {
		keybindings = KEYBINDINGS }
	aliases := this.KeyAliases
	if aliases == nil {
		aliases = KEY_ALIASES }
	// expand aliases...
	seen := []string{ key }
	if action, exists := keybindings[key] ; exists {
		_action := action
		for exists && ! slices.Contains(seen, _action) {
			if _action, exists = keybindings[_action] ; exists {
				seen = append(seen, _action)
				action = _action } }
		return this.HandleAction(action) }
	// call key alias...
	parts := strings.Split(key, "+")
	if aliases, exists := aliases[parts[len(parts)-1]] ; exists {
		for _, key := range aliases {
			res := this.HandleKey(
				strings.Join(append(parts[:len(parts)-1], key), "+"))
			if res == Missing {
				log.Println("Key Unhandled:",
					strings.Join(append(parts[:len(parts)-1], key), "+"))
				continue }
			return res } }
	return Missing }

func (this *TcellDrawer) Setup(lines Lines) *TcellDrawer {
	this.Lines = &lines
	this.Actions = NewActions(this)
	lines.CellsDrawer = this
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Panic(err) }
	this.Screen = screen
	if err := this.Screen.Init(); err != nil {
		log.Panic(err) }
	this.EnableMouse()
	this.EnablePaste()
	this.EnableFocus()
	return this }
// XXX can we detect mod key press???
//		...need to detect release of shift in selection...
// XXX add background fill...
// XXX might be fun to indirect this, i.e. add a global workspace manager
//		that would pass events to clients/windows and handle their draw 
//		order...
func (this *TcellDrawer) Loop() Result {
	defer this.Finalize()

	// initial state...
	this.
		updateGeometry().
		Draw()

	for {
		this.Show()

		evt := this.PollEvent()

		switch evt := evt.(type) {
			// geometry...
			case *tcell.EventResize:
				this.
					updateGeometry().
					Draw()
			// keys...
			case *tcell.EventKey:
				key_handled := false
				for _, key := range TcellEvent2Keys(*evt) {
					res := this.HandleKey(key)
					if res == Missing {
						log.Println("Key Unhandled:", key)
						continue }
					if res != OK {
						return res } 
					key_handled = true
					this.Draw()
					break }
				// do not check for deafults on keys we handled...
				if key_handled {
					continue }
				// defaults...
				if evt.Key() == tcell.KeyEscape || 
						evt.Key() == tcell.KeyCtrlC {
					return OK }
			// mouse...
			case *tcell.EventMouse:
				buttons := evt.Buttons()
				// get modifiers...
				// XXX this is almost the same as in TcellEvent2Keys(..) can we generalize???
				mod := evt.Modifiers()
				mods := []string{}
				if mod & tcell.ModCtrl != 0 {
					mods = append(mods, "ctrl") }
				if mod & tcell.ModAlt != 0 {
					mods = append(mods, "alt") }
				if mod & tcell.ModMeta != 0 {
					mods = append(mods, "meta") }
				if mod & tcell.ModShift != 0 {
					mods = append(mods, "shift") }
				// XXX handle double click...
				// XXX handle modifiers...
				if buttons & tcell.Button1 != 0 || 
						buttons & tcell.Button2 != 0 {
					//log.Println("CLICK:", mods)
					col, row := evt.Position()
					// ignore clicks outside the list...
					if col < this.Lines.Left || 
								col >= this.Lines.Left + this.Lines.Width || 
							row < this.Lines.Top || 
								row >= this.Lines.Top + this.Lines.Height {
						//log.Println("    OUT OF BOUNDS")
						continue }
					// title/status bars and borders...
					top_offset := 0
					if ! this.Lines.TitleDisabled {
						top_offset = 1
						if row == this.Lines.Top {
							// XXX handle titlebar click???
							//log.Println("    TITLE_LINE")
							continue } }
					if ! this.Lines.StatusDisabled {
						if row - this.Lines.Top == this.Lines.Rows() + 1 {
							// XXX handle statusbar click???
							//log.Println("    STATUS_LINE")
							continue } }
					if this.Lines.Border != "" {
						if col == this.Lines.Left ||
								(! this.Lines.Scrollable() && 
									col == this.Lines.Left + this.Lines.Width - 1) {
							//log.Println("    BORDER")
							continue } }
					// scrollbar...
					// XXX sould be nice if we started in the scrollbar 
					//		to keep handling the drag untill released...
					//		...for this to work need to either detect 
					//		drag or release...
					if this.Lines.Scrollable() && 
							col == this.Lines.Left + this.Lines.Width - 1 {
						//log.Println("    SCROLLBAR")
						this.Lines.RowOffset = 
							int((float64(row - this.Lines.Top - top_offset) / float64(this.Lines.Rows() - 1)) * 
							float64(len(this.Lines.Lines) - this.Lines.Rows()))
						this.Draw()
					// call click handler...
					} else {
						border := 0
						if this.Lines.Border != "" {
							border = 1 }
						this.MouseCol = col - this.Lines.Left - border
						this.MouseRow = row - this.Lines.Top
						if ! this.Lines.TitleDisabled {
							this.MouseRow-- }

						// empty space below rows...
						if this.MouseRow >= len(this.Lines.Lines) {
							if this.EmptySpace != "active" {
								//log.Println("    EMPTY SPACE")
								this.Lines.Index = len(this.Lines.Lines) - 1 }
							continue }

						button := ""
						// normalize buttons...
						if buttons & tcell.Button1 != 0 {
							button = "MouseLeft" }
						if buttons & tcell.Button2 != 0 {
							button = "MouseRight" }
						if buttons & tcell.Button3 != 0 {
							button = "MouseMiddle" }
						for _, key := range key2keys(mods, button) {
							res := this.HandleKey(key) 
							if res == Missing {
								continue }
							if res != OK {
								return res } 
							this.Draw()
							break } }
					this.handleScrollLimits()

				} else if buttons & tcell.WheelUp != 0 {
					// XXX add mods...
					res := this.HandleKey("WheelUp")
					if res == Missing {
						res = OK }
					if res != OK {
						return res }
					this.Draw()

				} else if buttons & tcell.WheelDown != 0 {
					// XXX add mods...
					res := this.HandleKey("WheelDown")
					if res == Missing {
						res = OK }
					if res != OK {
						return res } 
					this.Draw() } } }
	return OK }
// handle panics and cleanup...
func (this *TcellDrawer) Finalize() {
	maybePanic := recover()
	this.Screen.Fini()
	if maybePanic != nil {
		panic(maybePanic) } }


func (this *TcellDrawer) Append(str string) *TcellDrawer {
	if this.Transformer != nil {
		//log.Println("  append:", str)
		_, err := this.Transformer.Write(str +"\n")
		if err != nil {
			log.Panic(err) }
	} else {
		this.Lines.Append(str) }
	this.Refresh() 
	return this }


/* XXX not sure about the API yet...
func (this *TcellDrawer) Append(str string) *TcellDrawer {
	this.Lines.Append(str)
	// XXX do transform...
	return this }
//*/
// XXX BUG: this sometimes does not go through the whole list...
//		...does Run(..) close .Stdout too early???
func (this *TcellDrawer) ReadFromCmd(cmd string) chan bool {
	this.Lines.Clear()
	running := make(chan bool)
	c, err := Run(cmd, nil)
	if err != nil {
		log.Panic(err) }
	go func(){
		// load...
		scanner := bufio.NewScanner(c.Stdout)
		for scanner.Scan() {
			txt := scanner.Text()
			//log.Println("read:", txt)
			this.Append(txt) } 
			//this.Append(scanner.Text()) } 
		close(running) }()
	return running }
// XXX EXPERIMENTAL...
// XXX should we transform the existing lines???
func (this *TcellDrawer) TransformCmd(cmd string) *TcellDrawer {
	c, err := Pipe(cmd,
		func(str string){
			//log.Println("    updated:", str)
			this.Lines.Append(str) 
			this.Refresh() })
	if err != nil {
		log.Panic(err) }
	this.Transformer = c
	return this }


// XXX should this take Lines ot Settings???
func NewTcellLines(l ...Lines) *TcellDrawer {
	var lines Lines
	if len(l) == 0 {
		lines = Lines{}
	} else {
		lines = l[0] }

	drawer := TcellDrawer{}
	drawer.Setup(lines)

	return &drawer }




// XXX need to separate out stderr to the original tty as it messes up 
//		ui + keep it redirectable... 
func main(){
	//* XXX stub...
	lines := NewTcellLines(Lines{
		SpanMode: "*,8",
		SpanSeparator: "│",
		Border: "│┌─┐│└─┘",
		// XXX BUG: this loses the space at the end of $TEXT and draws 
		//		a space intead of "/"...
		Title: " $TEXT |/",
		Status: "|${SELECTED:!*}${SELECTED:+($SELECTED)}$F $LINE/$LINES ",
	})
	/* XXX
	lines.Lines.Append(
		"Some text",
		"Current|",
		"Some|Columns")
	for i := 0; i < 10; i++ {
		lines.Lines.Append(fmt.Sprint("bam|", i)) }
	lines.Lines.Index = 1
	lines.Lines.Lines[0].Selected = true
	/*/
	lines.TransformCmd("sed 's/$/|/'")
	// NOTE: ls flags that trigger stat make things really slow (-F, sorting, ...etc)
	//lines.ReadFromCmd("ls")
	lines.ReadFromCmd("echo .. ; ls -t --group-directories-first ~/Pictures/Instagram/")
	//lines.ReadFromCmd("echo .. ; ls --group-directories-first ~/Pictures/Instagram/ARCHIVE/")
	//*/

	//lines.Width = "50%"
	//lines.Align = []string{"right"}
	/*/
	lines := NewTcellLines()

	// XXX set settings...
	// XXX

	//*/

	os.Exit(
		toExitCode(
			lines.Loop())) }



// vim:set sw=4 ts=4 nowrap :
