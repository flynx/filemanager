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
	//"fmt"
	"log"
	"os"
	//"syscall"
	"reflect"
	"sync"
	"time"
	//"io"
	"bufio"
	"runtime"
	//"bytes"
	"strings"
	//"unicode"
	"strconv"
	"slices"
	//"regexp"

	"github.com/jessevdk/go-flags"
)


// command line args...
type Options struct {
	Pos struct {
		FILE string
	} `positional-args:"yes"`

	//ListCommand string `short:"c" long:"cmd" value-name:"CMD" env:"CMD" description:"List command"`
	// NOTE: this is not the same as filtering the input as it will be 
	//		done lazily when the line reaches view.
	//TransformCommand string `short:"t" long:"transform" value-name:"CMD" env:"TRANSFORM" description:"Row transform command"`
	TransformPopulateCommand string `short:"p" long:"transform-populate" value-name:"CMD" env:"TRANSFORM" description:"Row transform command"`

	SelectionCommand string `short:"e" long:"selection" value-name:"ACTION" env:"REJECT" description:"Command to filter selection from input"`

	// XXX doc: to match a number explicitly escape it with '\\'...
	Focus string `short:"f" long:"focus" value-name:"[N|STR]" env:"FOCUS" description:"Line number to focus"`
	FocusRow int `long:"focus-row" value-name:"N" env:"FOCUS_ROW" description:"Screen line number of focused line"`
	FocusCmd string `long:"focus-cmd" value-name:"CMD" env:"FOCUS_CMD" description:"Focus command"`

	/* XXX
	RowOffset int `long:"row-offset" value-name:"N" env:"ROW_OFFSET" description:"Row offset of visible lines"`
	//ColOffset int `long:"col-offset" value-name:"N" env:"COL_OFFSET" description:"Column offset of visible lines"`
	//*/

	// XXX
	//Selection: string ``
	//SelectionCmd: string ``

	// XXX chicken-egg: need to first parse the args then parse the ini 
	//		and then merge the two...
	//ArgsFile string `long:"args-file" value-name:"FILE" env:"ARGS" description:"Arguments file"`


	// Quick actions...
	Actions struct {
		Select string `short:"s" long:"select" value-name:"ACTION" env:"SELECT" description:"Action to execute on item select"`
		Reject string `short:"r" long:"reject" value-name:"ACTION" env:"REJECT" description:"Action to execute on reject"`
	} `group:"Actions"`

	Keyboard struct {
		Key map[string]string `short:"k" long:"key" value-name:"KEY:ACTION" description:"Bind key to action"`
	} `group:"Keyboard"`

	Chrome struct {
		Title string `long:"title" value-name:"STR" env:"TITLE" default:"%CMD%SPAN%SPINNER" description:"Title format"`
		TitleCommand string `long:"title-cmd" value-name:"CMD" env:"TITLE_CMD" description:"Title command"`
		Status string `long:"status" value-name:"STR" env:"STATUS" default:"%CMD%SPAN $LINE/$LINES " description:"Status format"`
		StatusCommand string `long:"status-cmd" value-name:"CMD" env:"STATUS_CMD" description:"Status command"`
		Size string `long:"size" value-name:"WIDTH,HEIGHT" env:"SIZE" default:"auto,auto" description:"Widget size"`
		Align string `long:"align" value-name:"LEFT,TOP" env:"ALIGN" default:"center,center" description:"Widget alignment"`
		Tab int `long:"tab" value-name:"COLS" env:"TABSIZE" default:"8" description:"Tab size"`
		Border bool `short:"b" long:"border" env:"BORDER" description:"Toggle border on"`
		//BorderChars string `long:"border-chars" env:"BORDER_CHARS" default:"│┌─┐│└─┘" description:"Border characters"`
		BorderChars string `long:"border-chars" env:"BORDER_CHARS" default:"single" description:"Border theme name or border characters"`
		SpinnerChars string `long:"spinner-chars" env:"SPINNER_CHARS" default:"10" description:"Spinner theme number or spinner characters"`
		Span string `long:"span" value-name:"[MODE|SIZE]" env:"SPAN" default:"fit-right" description:"Line spanning mode/size"`
		// XXX at this point this depends on leading '%'...
		//SpanMarker string `long:"span-marker" value-name:"STR" env:"SPAN_MARKER" default:"%SPAN" description:"Marker to use to span a line"`
		SpanExtend string `long:"span-extend" env:"SPAN_EXTEND" choice:"auto" choice:"always" choice:"never" default:"auto" description:"Extend span separator through unspanned and empty lines"`
		SpanSeparator string `long:"span-separator" value-name:"CHR" env:"SPAN_SEPARATOR" default:" " description:"Span separator character"`
		SpanLeftMin int `long:"span-left-min" value-name:"COLS" env:"SPAN_LEFT_MIN" default:"8" description:"Left column minimum span"`
		SpanRightMin int `long:"span-right-min" value-name:"COLS" env:"SPAN_RIGHT_MIN" default:"6" description:"Right column minimum span"`
		OverflowIndicator string `long:"overflow-indicator" value-name:"CHR" env:"OVERFLOW_INDICATOR" default:"}" description:"Line overflow character"`
		SpanFiller string `long:"span-filler" value-name:"CHR" env:"SPAN_FILLER" default:" " description:"Span fill character"`
		SpanFillerTitle string `long:"span-filler-title" value-name:"CHR" env:"SPAN_FILLER_TITLE" default:" " description:"Title span fill character"`
		SpanFillerStatus string `long:"span-filler-status" value-name:"CHR" env:"SPAN_FILLER_STATUS" default:" " description:"Status span fill character"`
		// XXX not sure what should be the default...
		EmptySpace string `long:"empty-space" choice:"passive" choice:"select-last" env:"EMPTY_SPACE" default:"passive" description:"Click in empty space below list action"`
	} `group:"Chrome"`

	Config struct {
		LogFile string `short:"l" long:"log" value-name:"FILE" env:"LOG" description:"Log file"`
		Separator string `long:"separator" value-name:"STRING" default:"\\n" env:"SEPARATOR" description:"Command separator"`
		// XXX might be fun to be able to set this to something like "middle"...
		ScrollThreshold int `long:"scroll-threshold" value-name:"N" default:"3" description:"Number of lines from the edge of screen to triger scrolling"`
		//ScrollThresholdTop int `long:"scroll-threshold-top" value-name:"N" default:"3" description:"Number of lines from the top edge of screen to triger scrolling"`
		//ScrollThresholdBottom int `long:"scroll-threshold-bottom" value-name:"N" default:"3" description:"Number of lines from the bottom edge of screen to triger scrolling"`
		// XXX add named themes/presets...
		//Theme map[string]string `long:"theme" value-name:"NAME:FGCOLOR:BGCOLOR" description:"Set theme color"`
	} `group:"Configuration"`

	Introspection struct {
		ListActions bool `long:"list-actions" description:"List available actions"`
		ListThemeable bool `long:"list-themeable" description:"List available themable element names"`
		ListBorderThemes bool `long:"list-border-themes" description:"List border theme names"`
		ListSpinners bool `long:"list-spinners" description:"List spinner styles"`
		ListColors bool `long:"list-colors" description:"List usable color names"`
	} `group:"Introspection"`

}

/* XXX
func HandleOptions() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return }
		log.Println("Error:", err)
		os.Exit(1) }

// 	// doc...
//	if options.Introspection.ListActions {
//		actions := Actions{}
//		t := reflect.TypeOf(&actions)
//		for i := 0; i < t.NumMethod(); i++ {
//			m := t.Method(i)
//			fmt.Println("    "+ m.Name) }
//		return }
//	if options.Introspection.ListThemeable {
//		for name, _ := range THEME {
//			fmt.Println("    "+ name) }
//		return }
//	if options.Introspection.ListBorderThemes {
//		names := []string{}
//		l := 0
//		for name, _ := range BORDER_THEME {
//			if len(name) > l {
//				l = len(name) }
//			names = append(names, name) }
//		slices.Sort(names)
//		for _, name := range names {
//			fmt.Printf("    %-"+ fmt.Sprint(l) +"v \"%v\"\n", name, BORDER_THEME[name]) }
//		return }
//	if options.Introspection.ListSpinners {
//		for i, style := range SPINNER_STYLES {
//			fmt.Printf("    %3v \"%v\"\n", i, style) }
//		return }
//	if options.Introspection.ListColors {
//		for name, _ := range tcell.ColorNames {
//			fmt.Println("    "+ name) }
//		return }

	lines := NewUI(
		Lines{
			SpanMode: "*,8",
			SpanSeparator: "│",
			Border: "│┌─┐│└─┘",
			// XXX BUG: this loses the space at the end of $TEXT and draws 
			//		a space intead of "/"...
			Title: " $TEXT |/",
			Status: "|${SELECTED:!*}${SELECTED:+($SELECTED)}$F $LINE/$LINES ",
		})
	lines.Options = options

// 	// globals...
//	INPUT_FILE = options.Pos.FILE
//	LIST_CMD = options.ListCommand
//	TRANSFORM_CMD = options.TransformCommand
//	TRANSFORM_POPULATE_CMD = options.TransformPopulateCommand
//	SELECTION_CMD = options.SelectionCommand

//	// focus/positioning...
//	FOCUS = options.Focus
//	CURRENT_ROW = options.FocusRow
//	FOCUS_CMD = options.FocusCmd

//	if options.Chrome.Border ||  
//			! parser.FindOptionByLongName("border-chars").IsSetDefault() {
//		BORDER = 1 
//		// char order: 
//		//		 01234567
//		//		"│┌─┐│└─┘"
//		// XXX might be fun to add border themes...
//		chars, ok := BORDER_THEME[options.Chrome.BorderChars]
//		border_chars := []rune{}
//		if ok {
//			border_chars = []rune(chars)
//		} else {
//			border_chars = []rune(
//				// normalize length...
//				fmt.Sprintf("%-8v", options.Chrome.BorderChars)) }
//		BORDER_LEFT = border_chars[0] 
//		BORDER_RIGHT = border_chars[4] 
//		BORDER_TOP = border_chars[2] 
//		BORDER_BOTTOM = border_chars[6] 
//		BORDER_CORNERS = map[string]rune{
//			"ul": border_chars[1],	
//			"ur": border_chars[3],	
//			"ll": border_chars[5],	
//			"lr": border_chars[7],	
//		} }

//	if i, err := strconv.Atoi(options.Chrome.SpinnerChars); err != nil {
//		SPINNER.Frames = options.Chrome.SpinnerChars
//	} else {
//		SPINNER.Frames = SPINNER_STYLES[i] }

// 	TITLE_LINE_FMT = options.Chrome.Title
//	TITLE_LINE = TITLE_LINE_FMT != ""
//	TITLE_CMD = options.Chrome.TitleCommand
//
//	STATUS_LINE_FMT = options.Chrome.Status
//	STATUS_LINE = STATUS_LINE_FMT != ""
//	STATUS_CMD = options.Chrome.StatusCommand
//
//	SIZE = strings.Split(options.Chrome.Size, ",")
//	ALIGN = strings.Split(options.Chrome.Align, ",")
//	TAB_SIZE = options.Chrome.Tab
//	SPAN_MODE = options.Chrome.Span
//	//SPAN_MARKER = options.Chrome.SpanMarker
//	SPAN_EXTEND = options.Chrome.SpanExtend
//	SPAN_LEFT_MIN_WIDTH = options.Chrome.SpanLeftMin
//	SPAN_RIGHT_MIN_WIDTH = options.Chrome.SpanRightMin
//	SPAN_FILLER = []rune(options.Chrome.SpanFiller)[0]
//	SPAN_FILLER_TITLE = []rune(fmt.Sprintf("%1v", options.Chrome.SpanFillerTitle))[0]
//	SPAN_FILLER_STATUS = []rune(fmt.Sprintf("%1v", options.Chrome.SpanFillerStatus))[0]
//	// defaults to SPAN_FILLER...
//	SPAN_SEPARATOR = SPAN_FILLER
//	if ! parser.FindOptionByLongName("span-separator").IsSetDefault() {
//		SPAN_SEPARATOR = []rune(fmt.Sprintf("%1v", options.Chrome.SpanSeparator))[0] }
//	OVERFLOW_INDICATOR = []rune(options.Chrome.OverflowIndicator)[0]
//	EMPTY_SPACE = options.Chrome.EmptySpace
//	// defaults to .ScrollThreshold...
//	SCROLL_THRESHOLD_TOP = options.Config.ScrollThreshold
//	if ! parser.FindOptionByLongName("scroll-threshold-top").IsSetDefault() {
//		SCROLL_THRESHOLD_TOP = options.Config.ScrollThresholdTop }
//	SCROLL_THRESHOLD_BOTTOM = options.Config.ScrollThreshold
//	if ! parser.FindOptionByLongName("scroll-threshold-bottom").IsSetDefault() {
//		SCROLL_THRESHOLD_BOTTOM = options.Config.ScrollThresholdBottom }
	
	// action aliases...
	if options.Actions.Select != "" {
		KEYBINDINGS["Select"] = 
			strings.ReplaceAll(
				options.Actions.Select, 
				options.Config.Separator, "\n") }
	if options.Actions.Reject != "" {
		KEYBINDINGS["Reject"] = 
			strings.ReplaceAll(
				options.Actions.Reject, 
				options.Config.Separator, "\n") }

	// keys...
	for key, action := range options.Keyboard.Key {
		KEYBINDINGS[key] = 
			strings.ReplaceAll(
				action, 
				options.Config.Separator, "\n") }

// 	// themes/colors...
//	for name, spec := range options.Config.Theme {
//		color := strings.SplitN(spec, ":", 2)
//		THEME[name] = 
//			tcell.StyleDefault.
//				Foreground(tcell.GetColor(color[0])).
//				Background(tcell.GetColor(color[1])) }

	// log...
	logFileName := options.Config.LogFile
	// XXX can we suppress only log.Print*(..) and keep errors and panic output???
	if logFileName == "" {
		logFileName = "/dev/null" }
	logFile, err := os.OpenFile(
		logFileName,
		os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	defer logFile.Close()
	if err != nil {
		log.Panic(err) }
	// Set log out put and enjoy :)
	log.SetOutput(logFile) 

	// output...
//	defer func(){
//		if STDOUT != "" {
//			fmt.Println(STDOUT) } }()

	// startup...
	os.Exit(
		toExitCode(
			lines.Loop())) }
//*/



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
	Setup(lines Lines)
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

	Lines *Lines
	Actions *Actions `no-flag:"true"`

	ListCommand string `short:"c" long:"cmd" value-name:"CMD" env:"CMD" description:"List command"`
	// NOTE: this is not the same as filtering the input as it will be 
	//		done lazily when the line reaches view.
	TransformCommand string `short:"t" long:"transform" value-name:"CMD" env:"TRANSFORM" description:"Row transform command"`

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

	ScrollThreshold int `long:"scroll-threshold" value-name:"N" default:"3" description:"Number of lines from the edge of screen to triger scrolling"`
	// Format:
	//		"passive" | "active"
	EmptySpace string `long:"empty-space" choice:"passive" choice:"select-last" env:"EMPTY_SPACE" default:"passive" description:"Click in empty space below list action"`

	Transformer *PipedCmd

	// caches...
	// NOTE: in normal use-cases the stuff cached here is static and 
	//		there should never be any leakage, if there is then something 
	//		odd is going on.
	__float_cache map[string]float64
	//__int_cache map[string]int

	RefreshInterval time.Duration
	__refresh_blocked sync.Mutex
	__refresh_pending sync.Mutex

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
func (this *UI) handleScrollLimits() *UI {
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
	this.Lines.CellsDrawer = this

	this.Renderer = drawer
	this.Renderer.Setup(lines)

	// XXX can we make these lazy or not-required???
	this.Actions = NewActions(this)

	return this }
func (this *UI) HandleArgs() Result {
	parser := flags.NewParser(this, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return Exit }
		os.Exit(1) } 
	return OK }
func (this *UI) Loop() Result {
	return this.Renderer.Loop(this) }

func (this *UI) Append(str string) *UI {
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
func (this *UI) Append(str string) *UI {
	this.Lines.Append(str)
	// XXX do transform...
	return this }
//*/
// XXX BUG: this sometimes does not go through the whole list...
//		...does Run(..) close .Stdout too early???
func (this *UI) ReadFromCmd(cmds ...string) chan bool {
	cmd := this.ListCommand
	if len(cmds) > 0 {
		cmd = cmds[0] }
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
// XXX add multiple transforms...
func (this *UI) TransformCmd(cmds ...string) *UI {
	cmd := this.TransformCommand
	if len(cmds) > 0 {
		cmd = cmds[0] }
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




// XXX need to separate out stderr to the original tty as it messes up 
//		ui + keep it redirectable... 
func main(){
	//* XXX stub...
	lines := NewUI(Lines{
		SpanMode: "*,8",
		SpanSeparator: "│",
		Border: "│┌─┐│└─┘",
		// XXX BUG: this loses the space at the end of $TEXT and draws 
		//		a space intead of "/"...
		Title: " $TEXT |/",
		Status: "|${SELECTED:!*}${SELECTED:+($SELECTED)}$F $LINE/$LINES ",
	})
	if lines.HandleArgs() == Exit {
		return }

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
	lines := NewUI()

	// XXX set settings...
	// XXX

	//*/

	os.Exit(
		toExitCode(
			lines.Loop())) }



// vim:set sw=4 ts=4 nowrap :
