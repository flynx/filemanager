/*
* XXX split this into two parts:
*		- tcell -- DONE
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
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"time"
	"io"
	"bufio"
	"runtime"
	//"bytes"
	"strings"
	"strconv"
	"slices"
	"maps"

	"github.com/jessevdk/go-flags"
)



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
	"Esc": "Reject",
	//"Esc": "Down",
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

func key2keys(mods []string, key string, rest ...string) [][]string {
	key_seq := [][]string{}
	Key := ""
	if len(rest) > 0 {
		Key = rest[0] }

	// XXX STUB -- still need 3 and 4 mod combinations for completeness...
	//		...generate combinations + sort by length...
	for i := 0; i < len(mods); i++ {
		for j := i+1; j < len(mods); j++ {
			key_seq = append(key_seq, []string{mods[i], mods[j], key}) } }
	for _, m := range mods {
		key_seq = append(key_seq, []string{m, key}) }
	// uppercase letter...
	if Key != "" {
		key_seq = append(key_seq, []string{Key}) }
	key_seq = append(key_seq, []string{key})

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

	// Skip handling this instance
	Skip
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
	*UI `no-flag:"true"`

	last string

	// can be:
	//	"select"
	//	"deselect"
	//	""			- toggle
	SelectMotion string
	SelectMotionStart int
}

func NewActions(d *UI) *Actions {
	return &Actions{
		UI: d,
	} }

// base action...
// update the .last attr with the action name...
func (this *Actions) Action() Result {
	pc, _, _, ok := runtime.Caller(1)
	this.last = ""
	if ok {
		path := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		this.last = path[len(path)-1] }
	//log.Println(this.last)
	return OK }

// Debug helper...
func (this *Actions) LOG() Result {
	log.Println("ACTION: LOG")
	return OK }

// General...
func (this *Actions) Focus() Result {
	// second click on same row...
	if this.MouseRow == this.Lines.Index - this.Lines.RowOffset {
		res := this.HandleKey("ClickSelected") 
		if res == Missing {
			res = OK }
		if res != OK {
			return res } }
	// select row...
	this.Lines.Index = this.MouseRow + this.Lines.RowOffset
	return OK }

// Vertical navigation...
func (this *Actions) Up() Result {
	this.Action()
	if this.Lines.Index - this.Lines.RowOffset > 0 && 
			// at scroll threshold (top)...
			(this.Lines.Index - this.Lines.RowOffset > this.ScrollThreshold ||
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
	if this.Lines.Index < len(this.Lines.Lines)-1 && 
			// within screen...
			this.Lines.Index - this.Lines.RowOffset < rows-1 && 
			// buffer smaller than screen...
			(rows >= len(this.Lines.Lines) ||
				// screen at end of buffer...
				this.Lines.RowOffset + rows == len(this.Lines.Lines) ||
				// at scroll threshold (bottom)...
				this.Lines.Index - this.Lines.RowOffset < rows - this.ScrollThreshold - 1) {
		this.Lines.Index++ 
	// scroll the buffer...
	} else {
		this.ScrollDown() }
	return OK }

// XXX should these also scroll focus???
//		...or should it be "dragged by screen"
func (this *Actions) ScrollUp() Result {
	this.Action()
	if this.Lines.RowOffset > 0 {
		this.Lines.RowOffset-- }
	if this.Lines.Index > 0 {
		this.Lines.Index-- }
	return OK }
func (this *Actions) ScrollDown() Result {
	this.Action()
	rows := this.Lines.Rows()
	if this.Lines.RowOffset + rows < len(this.Lines.Lines) {
		this.Lines.RowOffset++ } 
	if this.Lines.Index < len(this.Lines.Lines)-1 {
		this.Lines.Index++ }
	return OK }

// XXX should we keep focus position relative to screen???
func (this *Actions) PageUp() Result {
	this.Action()
	if this.Lines.RowOffset > 0 {
		rows := this.Lines.Rows()
		this.Lines.Index -= rows 
		if this.Lines.Index < 0 {
			this.Top() } 
	} else {
		this.Top() } 
	return OK }
func (this *Actions) PageDown() Result {
	this.Action()
	rows := this.Lines.Rows()
	if this.Lines.RowOffset < len(this.Lines.Lines) - rows {
		this.Lines.Index += rows 
		if this.Lines.Index >= len(this.Lines.Lines) {
			this.Bottom() }
	} else {
		this.Bottom() }
	return OK }

func (this *Actions) Top() Result {
	this.Action()
	this.Lines.Index = 0
	return OK }
func (this *Actions) Bottom() Result {
	this.Action()
	this.Lines.Index = len(this.Lines.Lines) - 1
	return OK }

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
		rows = []int{this.Lines.Index} }
	for _, i := range rows {
		this.Lines.Lines[i].Selected = true }
	return OK }
func (this *Actions) Deselect(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{this.Lines.Index} }
	for _, i := range rows {
		this.Lines.Lines[i].Selected = false }
	return OK }
// XXX us this.Lines.SelectToggle(..)
func (this *Actions) SelectToggle(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{this.Lines.Index} }
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
		if this.Lines.Lines[this.Lines.Index].Selected {
			this.SelectMotion = "deselect" } }
	log.Println("SELECTION", this.SelectMotion)
	this.Action()
	this.SelectMotionStart = this.Lines.Index
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
		end = this.Lines.Index
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
	return this.SelectEnd(this.Lines.Index) }

// Utility...
func (this *Actions) Update() Result {
	return this.UI.Update() }
func (this *Actions) Refresh() Result {
	this.UI.Refresh()
	return OK }

// XXX will this stop goroutines?? (TEST)
// XXX will this work on windows / windows version / disable on windows???
//		...how do we deal with things like cygwin/MinGW/..???
func (this *Actions) Stop() Result {
	this.UI.Renderer.Stop()
	return OK }
func (this *Actions) Fail() Result {
	return Fail }
func (this *Actions) Exit() Result {
	return Exit }



// Renderer...
//
type Renderer interface {
	drawCells(col, row int, str string, style_name string, style Style)

	ResetCache()

	Size() (width int, height int)

	Fill(style Style)
	Refresh()
	Setup(lines *Lines)
	Loop(this *UI) Result
	Stop()
	// XXX
	Finalize()
}



// UI...
//

var REFRESH_INTERVAL = time.Millisecond * 15

// XXX renmae...
type UI struct {
	Renderer `no-flag:"true"`

	// XXX watch/update on input filechange (.Files.Input)...
	//WatchFile bool

	ListCommand string `short:"c" long:"cmd" value-name:"CMD" env:"CMD" description:"List command"`
	Cmd *Cmd
	// NOTE: this is not the same as filtering the input as it will be 
	//		done lazily when the line reaches view.
	TransformCommand string `short:"t" long:"transform" value-name:"CMD" env:"TRANSFORM" description:"Row transform command"`
	Transformer *PipedCmd

	// XXX like transform but use output for selection...
	//SelectionCommand string `short:"e" long:"selection" value-name:"ACTION" env:"REJECT" description:"Command to filter selection from input"`

	// NOTE: these only affect startup -- set .Index and .RowOffset...
	// XXX these need a uniform startup-load mechanic done...
	Focus string `short:"f" long:"focus" value-name:"[N|STR]" env:"FOCUS" description:"Line number / line to focus"`
	//FocusRow int `long:"focus-row" value-name:"N" env:"FOCUS_ROW" description:"Screen line number of focused line"`
	//FocusCmd string `long:"focus-cmd" value-name:"CMD" env:"FOCUS_CMD" description:"Focus command"`
	JumpToFocus bool `long:"jump-to-focus" description:"If set jump to focus as soon as it is loaded without scrolling"`

	// Quick actions...
	Select string `short:"s" long:"select" value-name:"ACTION" env:"SELECT" description:"Action to execute on item select"`
	Reject string `short:"r" long:"reject" value-name:"ACTION" env:"REJECT" description:"Action to execute on reject"`

	// XXX revise naming...
	// XXX this is not seen by tcell...
	//FocusAction bool `long:"focus-action" description:"If not set the focusing click will be ignored"`


	Lines *Lines `group:"Chrome"`

	Actions *Actions `no-flag:"true"`

	// Keyboard...
	//
	KeyAliases KeyAliases
	KeybindingsDefaults Keybindings
	Keybindings Keybindings `short:"k" long:"key" value-name:"KEY:ACTION" description:"Bind key to action"`
	KeybindingsNoDefaults bool `long:"no-key-defaults" description:"Do not set default keybindings"`

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

	ScrollThreshold int `long:"scroll-threshold" value-name:"N" default:"3" description:"Number of lines from the edge of screen to triger scrolling"`
	EmptySpace string `long:"empty-space" choice:"passive" choice:"select-last" env:"EMPTY_SPACE" default:"passive" description:"Click in empty space below list action"`

	RefreshInterval time.Duration
	__refresh_blocked sync.Mutex
	__refresh_pending sync.Mutex

	__reading sync.Mutex
	__read_running chan bool
	__appending sync.Mutex
	__updating sync.Mutex

	__selection []string
	__focus string
	__index int

	// mark that stdin read was started...
	__stdin_read bool

	Introspection struct {
		ListActions bool `long:"list-actions" description:"List available actions"`
		ListThemeable bool `long:"list-themeable" description:"List available themable element names"`
		ListBorderThemes bool `long:"list-border-themes" description:"List border theme names"`
		ListSpinnerThemes bool `long:"list-spinners" description:"List spinner styles"`
		//ListColors bool `long:"list-colors" description:"List usable color names"`
		ListAll bool `long:"list-all" description:"List all"`
	} `group:"Introspection"`

	// XXX UGLY... 
	Files struct {
		Input string `positional-arg-name:"PATH"`
	} `positional-args:"true"`

	// caches...
	// NOTE: in normal use-cases the stuff cached here is static and 
	//		there should never be any leakage, if there is then something 
	//		odd is going on.
	__float_cache map[string]float64
	//__int_cache map[string]int
}

func (this *UI) ResetCache() *UI {
	this.__float_cache = nil
	//this.__int_cache = nil
	this.Renderer.ResetCache()
	return this }

// XXX TCELL uses .Screen.Size()
func (this *UI) updateGeometry() *UI {
	var err error
	W, H := this.Renderer.Size()
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
// XXX should we handle negative indexes here???
func (this *UI) handleScrollLimits() *UI {
	delta := 0

	rows := this.Lines.Rows()
	screen_focus_offset := this.Lines.Index - this.Lines.RowOffset
	top_threshold := this.ScrollThreshold
	bottom_threshold := rows - this.ScrollThreshold - 1 
	if rows < this.ScrollThreshold + this.ScrollThreshold {
		top_threshold = rows / 2
		bottom_threshold = rows - top_threshold }
	
	// buffer smaller than screen -- keep at top...
	if rows > len(this.Lines.Lines) {
		this.Lines.RowOffset = 0
		// normalize index...
		if this.Lines.Index > len(this.Lines.Lines) {
			this.Lines.Index = len(this.Lines.Lines)-1 }
		if this.Lines.Index < 0 {
			this.Lines.Index = 0 }
		return this }

	// keep from scrolling past the bottom of the screen...
	if this.Lines.RowOffset + rows > len(this.Lines.Lines) {
		delta = this.Lines.RowOffset - (len(this.Lines.Lines) - rows)
	// scroll to top threshold...
	} else if screen_focus_offset < top_threshold && 
			this.Lines.RowOffset > 0 {
		delta = top_threshold - screen_focus_offset
		if delta > this.Lines.RowOffset {
			delta = this.Lines.RowOffset }
	// keep current row on screen...
	} else if screen_focus_offset > bottom_threshold && 
			screen_focus_offset > top_threshold {
		delta = bottom_threshold - screen_focus_offset
		// saturate delta...
		if delta < (this.Lines.RowOffset + rows) - len(this.Lines.Lines) {
			delta = (this.Lines.RowOffset + rows) - len(this.Lines.Lines) } } 

	// do the update...
	if delta != 0 {
		this.Lines.RowOffset -= delta }

	return this }

func (this *UI) Fill() *UI {
	_, s := this.Lines.GetStyle("background")
	this.Renderer.Fill(s)
	return this }
func (this *UI) Draw() *UI {
	this.
		handleScrollLimits().
		// XXX do this separately...
		Fill().
		Lines.Draw()
	return this }
// NOTE: this will not refresh faster than once per .RefreshInterval
func (this *UI) Refresh() *UI {
	// refresh now...
	if this.__refresh_blocked.TryLock() {
		this.
			updateGeometry().
			Draw()
		this.Renderer.Refresh()
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
func (this *UI) HandleAction(actions string) Result {
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
func (this *UI) HandleKey(key string) Result {
	aliases := this.KeyAliases
	if aliases == nil {
		aliases = KEY_ALIASES }
	bindings := this.Keybindings
	if bindings == nil {
		bindings = KEYBINDINGS }
	// expand aliases...
	seen := []string{ key }
	if action, exists := bindings[key] ; exists {
		_action := action
		for exists && ! slices.Contains(seen, _action) {
			if _action, exists = bindings[_action] ; exists {
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
// XXX add zone key handling...
func (this *UI) HandleMouse(col, row int, pressed []string) Result {
	button := pressed[len(pressed)-1]
	switch button {
		case "MouseLeft", "MouseRight", "MouseMiddle":
			// ignore clicks outside the list...
			if col < this.Lines.Left || 
						col >= this.Lines.Left + this.Lines.Width || 
					row < this.Lines.Top || 
						row >= this.Lines.Top + this.Lines.Height {
				return Skip }
			// title/status bars and borders...
			top_offset := 0
			if ! this.Lines.TitleDisabled {
				top_offset = 1
				if row == this.Lines.Top {
					// XXX handle titlebar click???
					//log.Println("    TITLE_LINE")
					return Skip } }
			if ! this.Lines.StatusDisabled {
				if row - this.Lines.Top == this.Lines.Rows() + 1 {
					// XXX handle statusbar click???
					//log.Println("    STATUS_LINE")
					return Skip } }
			if this.Lines.Border != "" {
				if col == this.Lines.Left ||
						(! this.Lines.Scrollable() && 
							col == this.Lines.Left + this.Lines.Width - 1) {
					//log.Println("    BORDER")
					return Skip } }
			// scrollbar...
			// XXX sould be nice if we started in the scrollbar 
			//		to keep handling the drag untill released...
			//		...for this to work need to either detect 
			//		drag or release...
			if this.Lines.Scrollable() && 
					col == this.Lines.Left + this.Lines.Width - 1 {
				//log.Println("    SCROLLBAR")
				// XXX this seems to be broken by handleScrollLimits()...
				//		...either change .Index or add support for scrolling past it...
				this.Lines.RowOffset = 
					int(
						(float64(row - this.Lines.Top - top_offset) / 
							float64(this.Lines.Rows() - 1)) * 
						float64(len(this.Lines.Lines) - this.Lines.Rows()))
				i := this.Lines.RowOffset + int(float64(this.Lines.Rows() - 1) / 2)
				if i < 0 {
					i = 0 }
				if this.Lines.Index < this.Lines.RowOffset || 
						this.Lines.Index > this.Lines.RowOffset + this.Lines.Height {
					this.Lines.Index = i }
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
					if this.EmptySpace == "select-last" {
						//log.Println("    EMPTY SPACE")
						this.Lines.Index = len(this.Lines.Lines) - 1 }
					return Skip }

				defer this.handleScrollLimits()
				return this.HandleKey(strings.Join(pressed, "+")) }

		default:
			res := this.HandleKey(strings.Join(pressed, "+"))
			if res == Missing {
				res = OK }
			if res != OK {
				return res }
			this.Draw() }
	return OK }

func (this *UI) Setup(lines Lines, drawer Renderer) *UI {
	this.Lines = &lines
	// XXX can we not do this???
	this.Lines.CellsDrawer = this

	this.Renderer = drawer
	this.Renderer.Setup(&lines)

	this.Actions = NewActions(this)

	return this }
// XXX might be a good idea to move handlers to methods of specific structs...
func (this *UI) HandleArgs() Result {
	// parse...
	// XXX can we also parse the current .Renderer???
	parser := flags.NewParser(this, flags.Default)
	_, err := parser.Parse()

	// print help...
	if err != nil {
		if flags.WroteHelp(err) {
			return Exit }
		os.Exit(0) } 

	// introspection...
	introspection := []func(){}
	mapLister := func(m map[string]string) func() {
		return func(){
			if len(introspection) > 1 {
				fmt.Println("Border themes:") }
			order := []string{}
			for name, _ := range m {
				order = append(order, name) }
			slices.Sort(order)
			for _, name := range order {
				fmt.Printf("    %-20v %#v\n", name+":", m[name]) } } }
	// list actions...
	if this.Introspection.ListAll || 
			this.Introspection.ListActions {
		introspection = append(introspection,
			func(){
				if len(introspection) > 1 {
					fmt.Println("Actions:") }
				t := reflect.TypeOf(this.Actions)
				for i := 0; i < t.NumMethod(); i++ {
					fmt.Println("    "+ t.Method(i).Name) } }) }
	// list theamable...
	if this.Introspection.ListAll || 
			this.Introspection.ListThemeable {
		introspection = append(introspection,
			func(){
				if len(introspection) > 1 {
					fmt.Println("Theamable:") }
				for name, _ := range THEME {
					fmt.Println("    "+ name) } }) }
	// list borders...
	if this.Introspection.ListAll || 
			this.Introspection.ListBorderThemes {
		introspection = append(introspection, mapLister(BORDER_THEME)) }
	// list spinners...
	if this.Introspection.ListAll || 
			this.Introspection.ListSpinnerThemes {
		introspection = append(introspection, mapLister(SPINNER_THEME)) }
	if len(introspection) > 0 {
		for _, f := range(introspection) {
			f() 
			fmt.Println() }
		os.Exit(0) }

	// key binding...
	ensureKeybindings := func(bindings ...Keybindings) {
		// nothing to do...
		if len(bindings) == 1 && 
				bindings[0] != nil &&
				this.Keybindings != nil &&
				&bindings[0] == &this.Keybindings {
			return }
		kb := Keybindings{}
		// defaults...
		if ! this.KeybindingsNoDefaults {
			defaults := this.KeybindingsDefaults
			if defaults == nil {
				defaults = KEYBINDINGS }
			maps.Copy(kb, defaults) }
		// merge rest...
		for _, b := range bindings {
			if b != nil {
				maps.Copy(kb, b) } }
		this.Keybindings = kb }
	// merge key bindings...
	if this.Keybindings != nil {
		ensureKeybindings( 
			// force merge defaults...
			maps.Clone(this.Keybindings) ) 
	} else if this.KeybindingsDefaults != nil {
		ensureKeybindings() }
	// select...
	if this.Select != "" {
		ensureKeybindings(this.Keybindings)
		this.Keybindings["Select"] = this.Select }
	// reject...
	if this.Reject != "" {
		ensureKeybindings(this.Keybindings)
		this.Keybindings["Reject"] = this.Reject }

	// themes/colors...
	// XXX for some magical reason overloading "deafult" does not work...
	if this.Lines.Theme != nil {
		theme := Theme{}
		maps.Copy(theme, THEME)
		// parse colors...
		for name, style := range this.Lines.Theme {
			style = strings.Split(style[0], ",")
			for i, color := range style {
				style[i] = strings.TrimSpace(color) }
			this.Lines.Theme[name] = style }
		maps.Copy(theme, this.Lines.Theme)
		this.Lines.Theme = theme }
	// border...
	if theme, ok := BORDER_THEME[this.Lines.Border]; ok {
		this.Lines.Border = theme }
	// spinner...
	if theme, ok := SPINNER_THEME[this.Lines.Spinner.Frames]; ok {
		this.Lines.Spinner.Frames = theme }

	// load data...
	this.Update()
	return OK }
func (this *UI) Loop() Result {
	return this.Renderer.Loop(this) }

// XXX need a clean way to stop runinng commands....
//		...this is not clean yet...
// XXX rename...
func (this *UI) KillRunning() {
	if this.Transformer != nil {
		// XXX do we need thid???
		this.Transformer.Close()
		this.Transformer.Kill()
		this.Transformer = nil } 
	if this.Cmd != nil {
		this.Cmd.Kill()
		this.Cmd = nil } }

func (this *UI) ReadFrom(reader io.Reader) chan bool {
	// keep only one read running at a time...
	if ! this.__reading.TryLock() {
		return this.__read_running }
	running := make(chan bool)
	this.__read_running = running
	// prep the transform command if defined...
	this.TransformCmd()
	this.Lines.Reset()
	go func(){
		defer this.__reading.Unlock()
		defer close(running) 
		i := 0 
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			txt := scanner.Text()
			log.Println("---", txt)
			this.Append(txt) 
			i++ } 
		// trim lines...
		if i < len(this.Lines.Lines) {
			this.Lines.Lines = this.Lines.Lines[:i] } }()
	return running }
// XXX can we stop this??? 
func (this *UI) ReadFromFile(filename ...string) chan bool {
	name := this.Files.Input
	if len(filename) > 0 {
		name = filename[0] }
	if len(name) == 0 {
		c := make(chan bool)
		defer close(c)
		return c }
	file, err := os.Open(name)
	if err != nil {
		log.Fatal(err) }

	running := this.ReadFrom(file) 

	// cleanup...
	go func(){
		<-running
		file.Close() }()
	return running }
func (this *UI) ReadFromCmd(cmds ...string) chan bool {
	cmd := this.ListCommand
	if len(cmds) > 0 {
		cmd = cmds[0] }
	if len(cmd) == 0 {
		c := make(chan bool)
		defer close(c)
		return c }
	/* XXX need to stop the scanner if .Cmd is killed...
	c, err := Run(cmd, nil)
	/*/
	var err error
	var c *Cmd
	c, err = Run(cmd, 
		// XXX is this a good way to stop???
		func(str string) bool {
			if this.Cmd != c {
				return false }
			return true })
	//*/
	this.Cmd = c
	go func(){
		c.Wait()
		this.Cmd = nil }()
	if err != nil {
		log.Fatal(err) }

	return this.ReadFrom(c.Stdout) }

func (this *UI) AppendDirect(str string) int {
	this.__appending.Lock()
	defer this.__appending.Unlock()

	i := this.Lines.Append(str) 

	row := this.Lines.Lines[i]
	txt := row.Text
	if this.__selection != nil {
		for j, s := range this.__selection {
			if s != "" && txt == s {
				row.Selected = true
				if j == 0 {
					this.__selection = this.__selection[1:]
				} else {
					this.__selection[j] = "" } } } }
	if this.__focus != "" &&
			(this.__focus == txt ||
				// XXX regexp???
				strings.Contains(txt, this.__focus)) {
		this.__focus = ""
		this.__index = -1
		this.Lines.Index = i
	} else if this.__index == i {
		this.__index = -1
		this.Lines.Index = i 
	// move focus down untill we reach a target...
	} else if ! this.JumpToFocus &&
			(this.__focus != "" || 
				this.__index >= 0) {
		this.Lines.Index = i }

	return i }
func (this *UI) Append(str string) *UI {
	if this.Transformer != nil {
		_, err := this.Transformer.Write(str +"\n")
		if err != nil {
			return this }
			//log.Fatal(err) }
			//log.Panic(err) }
	} else {
		this.AppendDirect(str) }
	this.Refresh() 
	return this }

// XXX BUG: race: refreshing while a refresh is still ongoing may result
//		in remains of the interupted output in the current buffer...
//		to reproduce:
//			press ctrl-r in fast succession
func (this *UI) Update() Result {
	if this.__stdin_read {
		return OK }
	// XXX if this is used .KillRunning() becomdes not relevant... (???)
	if ! this.__updating.TryLock(){
		return OK }
	done := make(chan bool)
	close(done)
	// drop eerything already running...
	// XXX this should clear the pipes but sometimes we get leftovers of 
	//		running/killed commands...
	// XXX is this relevant with .__updating ???
	//		....yes, and the race is still here...
	this.KillRunning()
	//
	this.__selection = slices.Clone(this.Lines.Selected())
	this.__focus = this.Lines.Current()
	this.__index = -1
	// get focus from flag (initial run)...
	if this.Focus != "" {
		i, err := strconv.Atoi(this.Focus)
		if err != nil {
			this.__focus = this.Focus
		} else {
			this.__focus = ""
			this.__index = i-1 }
		this.Focus = "" }
	this.Lines.Spinner.Start()

	// file...
	if this.Files.Input != "" {
		done = this.ReadFromFile()
	// command...
	} else if this.ListCommand != "" {
		done = this.ReadFromCmd()
	// pipe...
	} else {
		// do this only once per run...
		this.__stdin_read = true
		stat, err := os.Stdin.Stat()
		if err != nil {
			log.Fatal(err) }
		if stat.Mode() & os.ModeNamedPipe != 0 {
			done = this.ReadFrom(os.Stdin) } } 

	go func(){
		<-done
		this.__updating.Unlock()
		this.Lines.Spinner.Stop() }()
	return OK }


// XXX EXPERIMENTAL...
// XXX might be a good idea to make this standalone/reusable (for selection/focus commands)
// XXX should we transform the existing lines???
// XXX add multiple transforms...
func (this *UI) TransformCmd(cmds ...string) *UI {
	if this.Transformer != nil {
		return this }
	cmd := this.TransformCommand
	if len(cmds) > 0 {
		cmd = cmds[0] }
	if len(cmd) == 0 {
		return this }
	var err error
	var c *PipedCmd
	c, err = Pipe(cmd,
		func(str string) bool {
			// stop if .Transformer killed/reset...
			if this.Transformer != c {
				return false }
			i := this.AppendDirect(str)
			this.Refresh() 
			if this.Lines.Length <= i {
				return false }
			return true })
	if err != nil {
		log.Fatal(err) }
	this.Transformer = c
	return this }


// XXX should this take Lines ot Settings???
func NewUI(l ...Lines) *UI {
	var lines Lines
	if len(l) == 0 {
		lines = Lines{}
	} else {
		lines = l[0] }

	ui := UI{}
	// XXX revise the Tcell passing...
	ui.Setup(lines, &Tcell{})

	return &ui }




// XXX BUG: this sometimes crashes:
//			ls | ./lines
//		running the same thing via -c seems ok...
// XXX need to separate out stderr to the original tty as it messes up 
//		ui + keep it redirectable... 
func main(){
	//* XXX stub...
	lines := NewUI()
	if lines.HandleArgs() == Exit {
		return }

	//lines.Width = "50%"
	//lines.Align = []string{"right"}

	os.Exit(
		toExitCode(
			lines.Loop())) }



// vim:set sw=4 ts=4 nowrap :
