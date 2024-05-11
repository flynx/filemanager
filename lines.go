/*
*
* Features:
*	- list with line navigation
*	- selection
*	- actions
*	- live search/filtering
*
* XXX BUG: partially failed command breaks the execution chain...
*		...e.g. for 'echo 123 ; ls moo' if ls fails the stdout will be 
*		smaller than expected and will break things...
*		...make the code tolerant to unexpected input...
* XXX BUG: keys are buffered and if a frame renders slowly all the 
*		buffered keys are handled... should be dropped...
*		...need a slow updating example/test...
* XXX BUG: scrollbar sometimes is off by 1 cell when scrolling down (small overflow)...
*		...can't reproduce...
*
*
* XXX might be a good idea to populate (-p) lines in the background rather 
*		than just on line access...
* XXX ASAP call tranform action in custom env where TEXT* vars refer to 
*		line being processed and not the focus line...
* XXX ASAP FOCUS_CMD need a way to set cursor position from command/action...
*		...e.g. in scripts/fileBrowser when going up one dir we need to
*		focus the old directory...
* XXX ASAP handle paste (and copy) -- actions...
* XXX might be good to add a hisotry mechanism letting the user store/access 
*		its state...
* XXX make aliases uniform -- usable anywhere an action can be used...
* XXX spinner...
* XXX would be nice to set width/height to fit content...
* XXX can we handle focus -- i.e. ignore first click if not focused...
* XXX move globals to struct and make the thing reusable (refactoring)...
* XXX for file argument, track changes to file and update... (+ option to disable)
* XXX might be good to handle title/status line clocks...
*		...preferably within the placeholders -- i.e. map cliks to %LINE, 
*		%LINES, $TEXT, ... this would require storing a map of the line...
* XXX can we load a screen with the curent terminal state as content???
*		modes:
*			inline (just after the current line)
*			floating
*			fill (current)
* XXX might be fun to add a stack of views...
*		...the top most one is shown and we can pop/push views to stack...
*		...this can be usefull to implement viewers and the like...
*		...this can also can be done by calling lines again...
* XXX might be fun to add an inline mode -- if # of lines is less that 
*		term height Println(..) them and then play with that region of 
*		the terminal, otherwise open normally...
* XXX concurent update + keep selection position/value...
* XXX might be fun to add border themes...
* XXX span overflow vs. line overflow is fixed in a hackish way -- revise...
*		would splitting at SPAN_MARKER and handling chunks be simpler??
*		see notes in: drawLine(..)
* XXX flags: can we set default values from variables???
*		...doing ` ... `+ VAR +` ... ` breaks things...
* XXX flags: formatting the config string breaks things...
*		e.g.:
*			ListCommand string `
*				short:"c" 
*				long:"cmd" 
*				value-name:"CMD" 
*				env:"CMD" 
*				description:"List command"`
*
*
* XXX button combinations to check if possible to handle uniquely in a term:
*		- ctrl-i (tab)
*		- shift+click
*		- ctrl-backspace
*
*/

package main

import "runtime"
import "os"
import "os/exec"
import "io"
import "sync"
import "time"
//import "path"
import "fmt"
import "log"
import "bytes"
import "slices"
import "strings"
import "strconv"
import "unicode"
import "bufio"
import "reflect"
import "regexp"

//import "go/importer"

import "github.com/jessevdk/go-flags"
import "github.com/gdamore/tcell/v2"


/*/ XXX refactoring -- not sure about this yet...
type Lines struct {
	TabSize uint
	Shell string

	Theme Theme

	Keybindings Keybindings

	// width, height
	Size []string
	// left, top
	Align []string

	ListCMD string
	TransformCMD string
	InputFile string
	// XXX should this be bytes.Buffer???
	Output string

	Left int
	Top int
	Width int
	Height int

	Cols uint
	Rows uint

	// text buffer...
	Text []Row

	TextWidth uint

	CurrentRow uint

	// screen offset within the .Text
	RowOffset uint
	ColOffset uint

	Scrollbar bool
	ScrollbarFG rune
	ScrollbarBG rune
	ScrollThreshold uint

	Actions Actions

	TitleLine bool
	TitleCmd string
	TitleLineFmt string

	StatusLine bool
	StatusCmd string
	StatusLineFmt string

	SpanMarker string
	SpanMode string
	SpanLeftMinWidth int
	SpanRightMinWidth int
	SpanSeparator rune
	OverflowIndicator rune
}
// XXX this part I do not like about Go -- no clean way to define the 
//		structure and te defaults in one place...
var LinesDefaults = Lines{
	TabSize: 8,
	Shell: "bash -c",
	Size: []string{"auto", "auto"},
	Align: []string{"center", "center"},
	ScrollbarFG: tcell.RuneCkBoard,
	ScrollbarBG: tcell.RuneBoard,
	ScrollThreshold: 3,

	TitleCmd: "",
	TitleLineFmt: "",

	StatusCmd: "",
	StatusLineFmt: "",

	SpanMarker: "%SPAN",
	SpanMode: "fit-right",
	SpanLeftMinWidth: 8,
	SpanRightMinWidth: 8,
	//SpanSeparator: tcell.RuneVLine,
	SpanSeparator: ' ',
	OverflowIndicator: '}',

	Theme: THEME,
	Keybindings: KEYBINDINGS,
	Actions: ACTIONS,
}


func New() Lines {
	copy := Lines(LinesDefaults)
	// XXX
	return copy }
//*/



var LIST_CMD string
var TRANSFORM_CMD string
var TRANSFORM_POPULATE_CMD string
var SELECTION_CMD string
var INPUT_FILE string
// XXX should this be a buffer???
var STDOUT string
//var STDERR string

// XXX need to account 
var SHELL = "bash -c"

var TAB_SIZE = 8

// width, height
var SIZE = []string{"auto", "auto"}
// left, top
var ALIGN = []string{"center", "center"}

var LEFT, TOP int
var WIDTH, HEIGHT int
// XXX rename...
var ROWS, COLS int
//var CONTENT_ROWS, CONTENT_COLS int
//var HOVER_COL, HOVER_ROW int

var MOUSE_COL int
var MOUSE_ROW int

var COL_OFFSET = 0
var ROW_OFFSET = 0

var BORDER = 0
var BORDER_LEFT = tcell.RuneVLine
var BORDER_RIGHT = tcell.RuneVLine
var BORDER_TOP = tcell.RuneHLine
var BORDER_BOTTOM = tcell.RuneHLine
var BORDER_CORNERS = map[string]rune{
	"ul": tcell.RuneULCorner,	
	"ur": tcell.RuneURCorner,	
	"ll": tcell.RuneLLCorner,	
	"lr": tcell.RuneLRCorner,	
}

var SCROLLBAR = 0
var SCROLLBAR_FG = tcell.RuneCkBoard
var SCROLLBAR_BG = tcell.RuneBoard

var SCROLL_THRESHOLD_TOP = 3
var SCROLL_THRESHOLD_BOTTOM = 3

var TITLE_CMD string
var TITLE_LINE = false
var TITLE_LINE_FMT = ""

var STATUS_CMD string
var STATUS_LINE = false
var STATUS_LINE_FMT = ""

var EMPTY_SPACE = "passive"

// XXX should this be '|' ???
var SPAN_MARKER = "%SPAN"
var SPAN_MODE = "fit-right"
var SPAN_EXTEND = "auto"
var SPAN_LEFT_MIN_WIDTH = 8
var SPAN_RIGHT_MIN_WIDTH = 8
var SPAN_FILLER = ' '
var SPAN_FILLER_TITLE = SPAN_FILLER
var SPAN_FILLER_STATUS = SPAN_FILLER
//var SPAN_SEPARATOR = tcell.RuneVLine
var SPAN_SEPARATOR = ' '

var OVERFLOW_INDICATOR = '}'

var FOCUS string
var FOCUS_CMD string

// current row relative to viewport...
var CURRENT_ROW = 0

// XXX cursor mode...
//		- cursor
//		- line
//		- page
//		- pattern


type Row struct {
	selected bool
	transformed bool
	populated bool
	text string
}
//type LinesBuffer []Row
type LinesBuffer struct {
	Use sync.Mutex
	Lines []Row
	Width int
}
func (this *LinesBuffer) Clear() *LinesBuffer {
	CURRENT_ROW = 0
	ROW_OFFSET = 0
	this.Lines = []Row{}
	this.Width = 0
	return this }
func (this *LinesBuffer) String() string {
	lines := []string{}
	for _, line := range this.Lines {
		lines = append(lines, line.text) }
	return strings.Join(lines, "\n") }
func (this *LinesBuffer) Push(line string) *LinesBuffer {
	this.Lines = append(this.Lines, Row{ text: line })
	l := len([]rune(line))
	if this.Width < l {
		this.Width = l }
	return this }
func (this *LinesBuffer) Append(str string) *LinesBuffer {
	for _, str := range strings.Split(str, "\n") {
		this.Push(str) } 
	return this }
func (this *LinesBuffer) AppendBuf(buf io.Reader) *LinesBuffer {
	scanner := bufio.NewScanner(buf)
	for scanner.Scan(){
		this.Push(scanner.Text()) }
	return this }
	// XXX should we unlock in the end or at current position/screen...
func (this *LinesBuffer) Write(str string) *LinesBuffer {
	this.Use.Lock()
	defer this.Use.Unlock()
	this.
		Clear().
		Append(str)
	return this }
	// XXX should we unlock in the end or at current position/screen...
func (this *LinesBuffer) WriteBuf(buf io.Reader) *LinesBuffer {
	this.Use.Lock()
	defer this.Use.Unlock()
	this.
		Clear().
		AppendBuf(buf)
	return this }

var TEXT_BUFFER LinesBuffer

var SELECTION = []string{}

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
	"q": "Reject",

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
	"shift+PageDn": `
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


// XXX should colors be stored as strings or as direct color values???
//		the binary (curent) form needs parsing on load (se arg handling) 
//		while the text will need parsing on use...
type Theme map[string]tcell.Style
var THEME = Theme {
	"default": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset),
	"current": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset).
		Reverse(true),
	"selected": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorYellow),
	"current-selected": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorYellow).
		Reverse(true),
	"span-separator": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorGray),
	"status-line": tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorReset),
	"status-span-separator": tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorReset),
	"title-line": tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorReset),
	"title-span-separator": tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorReset),
	"background": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset),
	"border": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset),
}

type BorderTheme map[string]string
var BORDER_THEME = BorderTheme {
	"single": "│┌─┐│└─┘",
	"thick": "┃┏━┓┃┗━┛",
	"double": "║╔═╗║╚═╝",
	"mixed": "│┌─┒┃┕━┛",
	"mixed-double": "│┌─╖║╘═╝",
	"single-double": "│╒═╕│╘═╛",
	"double-single": "║╓─╖║╙─╜",
	"shaded": "│┌─┐┃└━┛",
	"shaded-double": "│┌─┐║└═╝",
	"ascii": "|+-+|+-+",
}


// Spinner...
type Spinner struct {
	Frames string
	State int

	running int
	starting sync.Mutex
}
func (this *Spinner) String() string {
	if this.running <= 0 {
		return "" } 
	return string([]rune(this.Frames)[this.State]) }
func (this *Spinner) Start() {
	go func(){
		//this.starting.Lock()
		//defer this.starting.Unlock()
		if this.running > 0 {
			this.running++ 
			return }
		this.running++ 
		if this.State < 0 {
			this.Step() }
		go func(){
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				<-ticker.C
				if this.running <= 0 {
					return }
				this.Step() } }() }() }
func (this *Spinner) Stop() *Spinner {
	if this.running == 1 {
		return this.StopAll() }
	if this.running > 0 {
		this.running-- }
	return this }
func (this *Spinner) StopAll() *Spinner {
	if this.running > 0 {
		this.running = 0
		ACTIONS.Refresh() }
	return this }
// XXX should this draw the whole screen???
//		...might be nice to be able to only update the chrome (title/status)
func (this *Spinner) Step() string {
	this.State++
	if this.State >= len([]rune(this.Frames)) {
		this.State = 0 }
	// XXX should this draw the whole screen???
	ACTIONS.Refresh()
	return this.String() }
func (this *Spinner) Done() *Spinner {
	this.StopAll()
	return this }

var SPINNER_STYLES = []string{
	"┤┘┴└├┌┬┐",
	"⠁⠂⠄⡀⢀⠠⠐⠈",
	"⡀⡁⡂⡃⡄⡅⡆⡇⡈⡉⡊⡋⡌⡍⡎⡏⡐⡑⡒⡓⡔⡕⡖⡗⡘⡙⡚⡛⡜⡝⡞⡟⡠⡡⡢⡣⡤⡥⡦⡧⡨⡩⡪⡫⡬⡭⡮⡯⡰⡱⡲⡳⡴⡵⡶⡷⡸⡹⡺⡻⡼⡽⡾⡿⢀⢁⢂⢃⢄⢅⢆⢇⢈⢉⢊⢋⢌⢍⢎⢏⢐⢑⢒⢓⢔⢕⢖⢗⢘⢙⢚⢛⢜⢝⢞⢟⢠⢡⢢⢣⢤⢥⢦⢧⢨⢩⢪⢫⢬⢭⢮⢯⢰⢱⢲⢳⢴⢵⢶⢷⢸⢹⢺⢻⢼⢽⢾⢿⣀⣁⣂⣃⣄⣅⣆⣇⣈⣉⣊⣋⣌⣍⣎⣏⣐⣑⣒⣓⣔⣕⣖⣗⣘⣙⣚⣛⣜⣝⣞⣟⣠⣡⣢⣣⣤⣥⣦⣧⣨⣩⣪⣫⣬⣭⣮⣯⣰⣱⣲⣳⣴⣵⣶⣷⣸⣹⣺⣻⣼⣽⣾⣿",
	"⣾⣽⣻⢿⡿⣟⣯⣷",
	"⠋⠙⠚⠒⠂⠂⠒⠲⠴⠦⠖⠒⠐⠐⠒⠓⠋",
	"⢄⢂⢁⡁⡈⡐⡠",
	"⠁⠂⠄⡀⢀⠠⠐⠈",
	"⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏",
	"☱☲☴☲",
	"◰◳◲◱",
	"◇◈◆",
	"◐◓◑◒",
	"◴◷◶◵",
	"■□▪▫",
	"▌▀▐▄",
	"▖▘▝▗",
	"│┌─┒┃┕━┛",
	"-\\|/",
	"v<^>",
}
var SPINNER = Spinner {
	Frames: "■□▪▫",
}


// XXX add support for ansi escape sequences...
//		...as a minimum strip them out...
// XXX trying to cheat my way out of implementing this...
func ansi2style(seq string, style tcell.Style) tcell.Style {
	// sanity check...
	if string(seq[:2]) != "\x1B[" || seq[len(seq)-1] != 'm' {
		log.Println("Error non-color escape sequence: \"ESC"+ seq[1:] +"\"") 
		return style }
	// normalize...
	args := []int{}
	for _, i := range strings.Split(string(seq[2:len(seq)-1]), ";") {
		s, err := strconv.Atoi(i)
		if err != nil {
			log.Println("Error parsing escape sequence: \"ESC"+ seq[1:] +"\":", err) 
			return style }
		args = append(args, s) }

	// XXX
	log.Println("---", args)

	return style }

var isEnvVar = regexp.MustCompile(`(\$[a-zA-Z_]+|\$\{[a-zA-Z_]+\})`)
var isPlaceholder = regexp.MustCompile(`(%[a-zA-Z_]+|%\{[a-zA-Z_]+\})`)
func populateTemplateLine(str string, cmd string) string {
	// handle env variables...
	env := makeEnv()
	str = string(isEnvVar.ReplaceAllFunc(
		[]byte(str), 
		func(match []byte) []byte {
			// normalize...
			name := string(match[1:])
			if name[0] == '{' {
				name = string(name[1:len(name)-1]) }
			// get the value...
			if val, ok := env[name] ; ok {
				return []byte(val)
			} else {
				return []byte(os.Getenv(name)) }
			return []byte("") }))

	// handle placeholders...
	str = string(isPlaceholder.ReplaceAllFunc(
		[]byte(str), 
		func(match []byte) []byte {
			// normalize...
			name := string(match[1:])
			if name[0] == '{' {
				name = string(name[1:len(name)-1]) }
			var err error
			val := ""
			current := Row{}
			//if len(TEXT_BUFFER.Lines) != 0 {
			if len(TEXT_BUFFER.Lines) > CURRENT_ROW + ROW_OFFSET {
				current = TEXT_BUFFER.Lines[CURRENT_ROW + ROW_OFFSET] } 
			switch name {
				// this has to be handled later, when the string is 
				// otherwise complete...
				case string(SPAN_MARKER[1:]):
					val = SPAN_MARKER
				case "SPINNER":
					val = SPINNER.String()
				case "CMD":
					if cmd != "" {
						val, err = callTransform(cmd, str)
						if err != nil {
							val = "" } }
				case "SELECTED":
					val = ""
					if current.selected {
						val = "*" }
				case "SELECTED_COUNT":
					val = fmt.Sprint(len(SELECTION))
					if val == "0" {
						val = "" }
				case "REST":
					val = current.text[COLS:] }
			return []byte(val) }))
	return str }

var EXTEND_SEPARATOR_COL = -1
func drawLine(col, row, width int, 
		str string, 
		span_mode string, span_filler rune, span_separator rune, 
		base_style tcell.Style, separator_style tcell.Style){

	// XXX HACK -- would dealing with parts be simpler???
	parts := strings.SplitN(str, SPAN_MARKER, 2)
	for i, part := range parts {
		if len([]rune(part)) >= width-2 {
			parts[i] = string([]rune(part)[:width-2]) } }
	line := []rune(strings.Join(parts, SPAN_MARKER))

	//line := []rune(str)

	col_offset := 0
	buf_offset := 0
	for i := 0; i < width - col_offset; i++ {
		cur_col := i + col_offset
		screen_col := col + cur_col
		buf_col := i + buf_offset
		style := base_style

		// get rune...
		c := ' '
		if buf_col < len(line) {
			c = line[buf_col]
		// extend span separator...
		} else if SPAN_EXTEND != "never" && 
				cur_col == EXTEND_SEPARATOR_COL {
			style = separator_style
			c = SPAN_SEPARATOR }

		// overflow indicator...
		if buf_col + col_offset == width - 1 && 
				buf_col < len(line)-1 {
			SCREEN.SetContent(screen_col, row, OVERFLOW_INDICATOR, nil, style)
			continue } 

		// escape sequences...
		// see: 
		//	https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797 
		if c == '\x1B' {
			// handle multiple adjacent sequences...
			for c == '\x1B' {
				j := buf_col + 1
				if line[j] == '[' {
					ansi_commands := "HfABCDEFGnsuJKmhlp"
					for j < len(line) && 
							! strings.ContainsRune(ansi_commands, line[j]) {
						j++ }
					// XXX handle color...
					//if line[j] == 'm' {
					//	style = ansi2style(string(line[buf_col:j+1]), style) }
				} else {
					ansi_direct_commands := "M78"
					for j < len(line) && 
							! strings.ContainsRune(ansi_direct_commands, line[j]) {
						j++ } } 
				buf_offset += (j + 1) - buf_col
				buf_col = j + 1
				if buf_col >= len(line) {
					c = ' ' 
				} else {
					c = line[buf_col] } }
			i--
			continue }

		// "%SPAN" -- expand/contract line to fit width...
		if c == '%' && 
				string(line[buf_col:buf_col+len(SPAN_MARKER)]) == SPAN_MARKER {
			offset := 0
			// automatic -- align to left/right edges...
			// NOTE: this essentially rigth-aligns the right side, it 
			//		will not attempt to left-align to the SPAN_SEPARATOR...
			// XXX should we attempty to draw a sraight vertical line between columns???
			if span_mode == "fit-right" {
				if len(line) - buf_col + SPAN_LEFT_MIN_WIDTH < width {
					offset = width - len(line)
				} else {
					offset = -buf_col + SPAN_LEFT_MIN_WIDTH }
			// manual...
			} else {
				c := 0
				// %...
				if span_mode[len(span_mode)-1] == '%' {
					// XXX parse this once...
					p, err := strconv.ParseFloat(string(span_mode[0:len(span_mode)-1]), 64)
					if err != nil {
						log.Println("Error parsing:", span_mode) }
					c = int(float64(width) * (p / 100))
					// normalize...
					if c < SPAN_LEFT_MIN_WIDTH {
						c = SPAN_LEFT_MIN_WIDTH }
					if width - c < SPAN_RIGHT_MIN_WIDTH {
						c = width - SPAN_RIGHT_MIN_WIDTH }
					if width < SPAN_LEFT_MIN_WIDTH + SPAN_RIGHT_MIN_WIDTH {
						r := float64(SPAN_LEFT_MIN_WIDTH) / float64(SPAN_RIGHT_MIN_WIDTH) 
						c = int(float64(width) * r) }
				// cols...
				} else {
					v, err := strconv.Atoi(span_mode) 
					if err != nil {
						log.Println("Error parsing:", span_mode) 
						continue }
					if v < 0 {
						c = width + v
					} else {
						c = v } }
				offset = c - buf_col - len(SPAN_MARKER) }
			// fill offset...
			for j := screen_col ; j < screen_col + offset + len(SPAN_MARKER) && j < col + width ; j++ {
				SCREEN.SetContent(j, row, span_filler, nil, style) } 
			// separator/overflow...
			if i + offset + len(SPAN_MARKER) <= width { 
				sep := span_separator
				if offset - col_offset + len(SPAN_MARKER) - 1 < 0 {
					sep = OVERFLOW_INDICATOR }
				EXTEND_SEPARATOR_COL = i + offset + len(SPAN_MARKER) - 1
				SCREEN.SetContent(col + EXTEND_SEPARATOR_COL, row, sep, nil, separator_style) 
			} else {
				// XXX is EXTEND_SEPARATOR_COL correct here?
				//		...can we reach this point BEFORE setting it???
				SCREEN.SetContent(col + EXTEND_SEPARATOR_COL, row, OVERFLOW_INDICATOR, nil, separator_style) }
			col_offset = offset
			// skip the marker...
			i += len(SPAN_MARKER) - 1
			continue }

		// tab -- offset output to next tabstop... 
		if c == '\t' { 
			// NOTE: the -1 here is to compensate fot the removed '\t'...
			offset := TAB_SIZE - ((buf_col + col_offset) % TAB_SIZE) - 1
			i := 0
			for ; i <= offset && cur_col + i < width ; i++ {
				SCREEN.SetContent(screen_col + i, row, ' ', nil, style) }
			// overflow indicator...
			if cur_col + i >= width {
				SCREEN.SetContent(screen_col + i - 1, row, OVERFLOW_INDICATOR, nil, style) }
			col_offset += offset 
			continue }

		// draw the rune...
		SCREEN.SetContent(screen_col, row, c, nil, style) } }


// populate the lines...
// XXX skip populated lines...
// XXX exit when text changes...
// XXX see population code below...
func populateLines(){
}

// XXX how do we handle borders when title/status does not contain %SPAN
//			$ ls | ./lines --title ' moo ' --border
//		vs:
//			$ ls | ./lines --title ' moo %SPAN' --border
func drawScreen(screen tcell.Screen, theme Theme){
	screen.Clear()
	TEXT_BUFFER.Use.Lock()
	defer TEXT_BUFFER.Use.Unlock()

	// scrollbar...
	var scroller_size, scroller_offset int
	scroller_style, ok := theme["scrollbar"]
	if ! ok {
		scroller_style = theme["default"] }
	if len(TEXT_BUFFER.Lines) > ROWS {
		SCROLLBAR = 1
	} else {
		SCROLLBAR = 0 }
	if SCROLLBAR > 0 {
		r := float64(ROWS) / float64(len(TEXT_BUFFER.Lines))
		scroller_size = 1 + int(float64(ROWS - 1) * r)
		scroller_offset = int(float64(ROW_OFFSET + 1) * r) }

	// set initial focus...
	if FOCUS != "" {
		f := 0
		// number...
		if i, err := strconv.Atoi(FOCUS); err == nil {
			f = i
		// string...
		} else {
			if FOCUS[0] == '\\' {
				FOCUS = string(FOCUS[1:]) }
			// XXX might also be a good idea to match content (and other) 
			//		then select best match...
			for i, r := range TEXT_BUFFER.Lines {
				if r.text == FOCUS {
					f = i
					break 
				// match parts/spans...
				} else if strings.Contains(r.text, SPAN_MARKER) {
					match := false
					for _, span := range strings.Split(r.text, SPAN_MARKER) {
						if strings.TrimSpace(span) == FOCUS {
							f = i
							match = true
							break } }
					if match {
						break } } } }
		// negative values...
		if f < 0 {
			f = len(TEXT_BUFFER.Lines) + f }
		// normalize...
		if f < 0 {
			f = 0 }
		if f > len(TEXT_BUFFER.Lines) {
			f = len(TEXT_BUFFER.Lines) - 1 }
		// set...
		if len(TEXT_BUFFER.Lines) < ROWS {
			ROW_OFFSET = 0
			CURRENT_ROW = f
		} else {
			ROW_OFFSET = f - CURRENT_ROW }
		// XXX revise -- can we overflow here???
		if ROW_OFFSET + CURRENT_ROW >= len(TEXT_BUFFER.Lines) {
			CURRENT_ROW = len(TEXT_BUFFER.Lines) - ROW_OFFSET }
		FOCUS = "" }

	// chrome detail themeing...
	separator_style, ok := theme["span-separator"]
	if ! ok {
		separator_style = theme["default"] }
	/* XXX do we need these??
	title_separator_style, ok := theme["title-span-separator"]
	if ! ok {
		title_separator_style, ok = theme["title-line"] 
		if ! ok {
			title_separator_style = theme["default"] } }
	status_separator_style, ok := theme["status-span-separator"]
	if ! ok {
		status_separator_style, ok = theme["status-line"] 
		if ! ok {
			status_separator_style = theme["default"] } }
	//*/
	border_style, ok := theme["border"]
	if ! ok {
		border_style = theme["default"] }

	row := TOP
	rows := HEIGHT
	cols := COLS
	style := theme["default"]

	left_offset := BORDER
	right_offset := BORDER
	if SCROLLBAR > 0 && BORDER < 1 {
		right_offset = SCROLLBAR }

	span_filler_title := SPAN_FILLER_TITLE
	span_filler_status := SPAN_FILLER_STATUS
	if BORDER > 0 {
		span_filler_title = BORDER_TOP
		span_filler_status = BORDER_BOTTOM }


	// title...
	EXTEND_SEPARATOR_COL = -1
	if TITLE_LINE {
		title_style := theme["title-line"]
		pre, post := "", ""
		if BORDER > 0 {
			pre, post = string(BORDER_CORNERS["ul"]), string(BORDER_CORNERS["ur"])
			title_style = border_style } 
		drawLine(LEFT, row, COLS, 
			pre + populateTemplateLine(TITLE_LINE_FMT, TITLE_CMD) + post, 
			"fit-right", span_filler_title, span_filler_title, 
			title_style, title_style) 
		rows--
		row++ }
	if STATUS_LINE {
		rows-- }
	// buffer...
	var populating sync.WaitGroup
	didPopulate := false
	for i := 0 ; i < rows ; i++ {
		separator_style := separator_style
		buf_row := i + ROW_OFFSET 

		line := &Row{}
		if buf_row < len(TEXT_BUFFER.Lines) {
			line = &TEXT_BUFFER.Lines[buf_row] }

		// theme...
		missing_style := false
		if buf_row >= 0 && 
				buf_row < len(TEXT_BUFFER.Lines) {
			// current+selected...
			style, missing_style = theme["default"] 
			if TEXT_BUFFER.Lines[buf_row].selected &&
					i == CURRENT_ROW {
				style, missing_style = theme["current-selected"]
				separator_style = style
			// current...
			} else if i == CURRENT_ROW {
				style, missing_style = theme["current"]
				separator_style = style
			// mark selected...
			} else if TEXT_BUFFER.Lines[buf_row].selected {
				style, missing_style = theme["selected"] } }
		// set default style...
		if ! missing_style {
			style = theme["default"] }

		// border vertical...
		if BORDER > 0 {
			screen.SetContent(LEFT, row, BORDER_LEFT, nil, border_style) }

		// line...
		if line.text != "" {
			if TRANSFORM_CMD != "" && 
					! line.transformed {
				line.transformed = true 
				str, err := callTransform(TRANSFORM_CMD, line.text)
				if err != nil {
					str = line.text }
				line.text = str } 
			if TRANSFORM_POPULATE_CMD != "" &&
					! line.populated {
				didPopulate = true
				populating.Add(1)
				cmd := goCallTransform(TRANSFORM_POPULATE_CMD, line.text)
				go func(){
					// skip if already populated or...
					if line.populated == true || 
							// TEXT_BUFFER changed...
							buf_row >= len(TEXT_BUFFER.Lines) ||
							line != &TEXT_BUFFER.Lines[buf_row] {
						return }
					defer populating.Done()
					scanner := bufio.NewScanner(*cmd.Stdout)
					lines := []string{}
					for scanner.Scan() {
						lines = append(lines, scanner.Text()) }
					line.populated = true 
					line.text = strings.Join(lines, "\n") }() } }
		drawLine(LEFT + BORDER, row, cols - left_offset - right_offset, 
			line.text, 
			SPAN_MODE, SPAN_FILLER, SPAN_SEPARATOR, 
			style, separator_style) 

		// border verticl...
		if BORDER > 0 && SCROLLBAR < 1 {
			screen.SetContent(LEFT + cols - 1, row, BORDER_RIGHT, nil, border_style)
		// scrollbar...
		} else if SCROLLBAR > 0 {
			c := SCROLLBAR_BG
			if i >= scroller_offset && 
					i < scroller_offset + scroller_size {
				c = SCROLLBAR_FG }
			screen.SetContent(LEFT + cols - 1, row, c, nil, scroller_style) }
		row++ }
	// async update populated lines...
	if didPopulate {
		go func(){
			populating.Wait()
			drawScreen(screen, theme) 
			screen.Sync() }() }

	// status...
	EXTEND_SEPARATOR_COL = -1
	if STATUS_LINE {
		status_style := theme["status-line"]
		pre, post := "", ""
		if BORDER > 0 {
			pre, post = string(BORDER_CORNERS["ll"]), string(BORDER_CORNERS["lr"])
			status_style = border_style } 
		drawLine(LEFT, row, COLS, 
			pre + populateTemplateLine(STATUS_LINE_FMT, STATUS_CMD) + post, 
			"fit-right", span_filler_status, span_filler_status, 
			status_style, status_style) } }


// Actions...
type Actions struct {
	last string
}

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

// base action...
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

func (this *Actions) Focus() Result {
	// second click on same row...
	if MOUSE_ROW == CURRENT_ROW {
		res := callHandler("ClickSelected") 
		if res == Missing {
			res = OK }
		if res != OK {
			return res } }
	// select row...
	CURRENT_ROW = MOUSE_ROW 
	return OK }

// vertical navigation...
func (this *Actions) Up() Result {
	this.Action()
	if CURRENT_ROW > 0 && 
			// account for SCROLL_THRESHOLD_TOP...
			(CURRENT_ROW > SCROLL_THRESHOLD_TOP ||
				ROW_OFFSET == 0) {
		CURRENT_ROW-- 
	// scroll the buffer...
	} else {
		this.ScrollUp() }
	return OK }
func (this *Actions) Down() Result {
	this.Action()
	// within the text buffer...
	if CURRENT_ROW + ROW_OFFSET < len(TEXT_BUFFER.Lines)-1 && 
			// within screen...
			CURRENT_ROW < ROWS-1 && 
			// buffer smaller than screen...
			(ROWS >= len(TEXT_BUFFER.Lines) ||
				// screen at end of buffer...
				ROW_OFFSET + ROWS == len(TEXT_BUFFER.Lines) ||
				// at scroll threshold...
				CURRENT_ROW < ROWS - SCROLL_THRESHOLD_BOTTOM - 1) {
		CURRENT_ROW++ 
	// scroll the buffer...
	} else {
		this.ScrollDown() }
	return OK }

// XXX should these track CURRENT_ROW relative to screen (current) or 
//		relative to content???
func (this *Actions) ScrollUp() Result {
	this.Action()
	if ROW_OFFSET > 0 {
		ROW_OFFSET-- }
	return OK }
func (this *Actions) ScrollDown() Result {
	this.Action()
	if ROW_OFFSET + ROWS < len(TEXT_BUFFER.Lines) {
		ROW_OFFSET++ } 
	return OK }

func (this *Actions) PageUp() Result {
	this.Action()
	if ROW_OFFSET > 0 {
		ROW_OFFSET -= ROWS 
		if ROW_OFFSET < 0 {
			this.Top() } 
	} else if ROW_OFFSET == 0 {
		this.Top() } 
	return OK }
func (this *Actions) PageDown() Result {
	this.Action()
	if len(TEXT_BUFFER.Lines) < ROWS {
		CURRENT_ROW = len(TEXT_BUFFER.Lines) - 1
		return OK }
	offset := len(TEXT_BUFFER.Lines) - ROWS
	if ROW_OFFSET < offset {
		ROW_OFFSET += ROWS 
		if ROW_OFFSET > offset {
			this.Bottom() } 
	} else if ROW_OFFSET == offset {
		this.Bottom() } 
	return OK }

func (this *Actions) Top() Result {
	this.Action()
	if ROW_OFFSET == 0 {
		CURRENT_ROW = 0 
	} else {
		ROW_OFFSET = 0 }
	return OK }
func (this *Actions) Bottom() Result {
	this.Action()
	if len(TEXT_BUFFER.Lines) < ROWS {
		CURRENT_ROW = len(TEXT_BUFFER.Lines) - 1
		return OK }
	offset := len(TEXT_BUFFER.Lines) - ROWS 
	if ROW_OFFSET == offset {
		CURRENT_ROW = ROWS - 1
	} else {
		ROW_OFFSET = len(TEXT_BUFFER.Lines) - ROWS }
	return OK }

/*// XXX horizontal navigation...
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

// selection...
func GetSelection() []string {
	selection := []string{}
	for _, row := range TEXT_BUFFER.Lines {
		if row.selected {
			selection = append(selection, row.text) } }
	return selection }
// NOTE: the selection is expected to mostly be in order.
// XXX would be nice to be able to match only left/right side of span...
//		...not sure how to configure this...
func SetSelection(selection []string){
	SELECTION = []string{}
	// clear old selection...
	// NOTE: we can't avoid this loop as doing this in the main loop can 
	//		potentially mess up already found results...
	for _, row := range TEXT_BUFFER.Lines {
		row.selected = false }
	var i = 0
	for _, line := range selection {
		for i < len(TEXT_BUFFER.Lines) {
			if line == TEXT_BUFFER.Lines[i].text {
				TEXT_BUFFER.Lines[i].selected = true 
				SELECTION = append(SELECTION, TEXT_BUFFER.Lines[i].text) } 
			i++ }
		// loop over TEXT_BUFFER in case we've got the selection out of 
		// order...
		if i >= len(TEXT_BUFFER.Lines) - 1 {
			i = 0 } } }
func updateSelectionList(){
	SELECTION = GetSelection() }
func (this *Actions) Select(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{CURRENT_ROW + ROW_OFFSET} }
	for _, i := range rows {
		TEXT_BUFFER.Lines[i].selected = true }
	updateSelectionList()
	return OK }
func (this *Actions) Deselect(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{CURRENT_ROW + ROW_OFFSET} }
	for _, i := range rows {
		TEXT_BUFFER.Lines[i].selected = false }
	updateSelectionList()
	return OK }
func (this *Actions) SelectToggle(rows ...int) Result {
	this.Action()
	if len(rows) == 0 {
		rows = []int{CURRENT_ROW + ROW_OFFSET} }
	for _, i := range rows {
		if TEXT_BUFFER.Lines[i].selected {
			TEXT_BUFFER.Lines[i].selected = false 
		} else {
			TEXT_BUFFER.Lines[i].selected = true } }
	updateSelectionList()
	return OK }
func (this *Actions) SelectAll() Result {
	this.Action()
	for i := 0; i < len(TEXT_BUFFER.Lines); i++ {
		TEXT_BUFFER.Lines[i].selected = true } 
	updateSelectionList()
	return OK }
func (this *Actions) SelectNone() Result {
	this.Action()
	for i := 0; i < len(TEXT_BUFFER.Lines); i++ {
		TEXT_BUFFER.Lines[i].selected = false } 
	SELECTION = []string{}
	return OK }
func (this *Actions) SelectInverse() Result {
	this.Action()
	rows := []int{}
	for i := 0 ; i < len(TEXT_BUFFER.Lines) ; i++ {
		rows = append(rows, i) }
	return this.SelectToggle(rows...) }
// can be:
//	"select"
//	"deselect"
//	""			- toggle
var SELECT_MOTION = ""
var SELECT_MOTION_START int
// XXX should these be usable standalone???
// XXX not sure how/if to set toggle mode....
func (this *Actions) SelectStart() Result {
	if this.last != "SelectEnd" {
		log.Println("NEW SELECTION", this.last)
		SELECT_MOTION = "select"
		if TEXT_BUFFER.Lines[CURRENT_ROW+ROW_OFFSET].selected {
			SELECT_MOTION = "deselect" } }
	log.Println("SELECTION", SELECT_MOTION)
	this.Action()
	SELECT_MOTION_START = CURRENT_ROW + ROW_OFFSET
	return OK }
func (this *Actions) SelectEnd(rows ...int) Result {
	this.Action()
	var start, end int
	if len(rows) >= 2 {
		start, end = rows[0], rows[1] 
	} else if len(rows) == 1 {
		start, end = SELECT_MOTION_START, rows[0]
	} else {
		start = SELECT_MOTION_START
		// NOTE: we are not selecting the last item intentionally...
		end = CURRENT_ROW + ROW_OFFSET - 1 }
	// normalize direction...
	if SELECT_MOTION_START > end {
		start, end = end, start }
	lines := []int{}
	for i := start ; i <= end; i++ {
		lines = append(lines, i) }
	if SELECT_MOTION == "select" {
		this.Select(lines...)
	} else if SELECT_MOTION == "deselect" {
		this.Deselect(lines...)
	} else {
		this.SelectToggle(lines...) }
	this.Action()
	return OK }
func (this *Actions) SelectEndCurrent() Result {
	return this.SelectEnd(CURRENT_ROW + ROW_OFFSET) }

// utility...
// XXX revise behaviour of reupdates on pipe...
func (this *Actions) Update() Result {
	selection := GetSelection()
	res := OK
	// file...
	if INPUT_FILE != "" {
		file, err := os.Open(INPUT_FILE)
		if err != nil {
			fmt.Println(err)
			return Fail }
		defer file.Close()
		TEXT_BUFFER.WriteBuf(file) 
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
			TEXT_BUFFER.WriteBuf(os.Stdin) } }
	SetSelection(selection)
	if FOCUS_CMD != "" {
		// XXX generate FOCUS
	}
	//this.Refresh()
	return res }
func (this *Actions) Refresh() Result {
	//SCREEN.Sync()
	drawScreen(SCREEN, THEME)
	SCREEN.Show()
	return OK }

func (this *Actions) Fail() Result {
	return Fail }
func (this *Actions) Exit() Result {
	return Exit }

var ACTIONS = Actions{}


var ENV = map[string]string {}
func getActiveList() []string {
	if len(SELECTION) == 0 && 
			len(TEXT_BUFFER.Lines) > 0 {
		return []string{TEXT_BUFFER.Lines[ROW_OFFSET + CURRENT_ROW].text} }
	return SELECTION }
func makeEnv() map[string]string {
	// pass data to command via env...
	selected := ""
	text := ""
	// vars we need text for...
	if len(TEXT_BUFFER.Lines) > CURRENT_ROW + ROW_OFFSET { 
		if TEXT_BUFFER.Lines[CURRENT_ROW + ROW_OFFSET].selected {
			selected = "*" }
		text = TEXT_BUFFER.Lines[CURRENT_ROW + ROW_OFFSET].text }
	env := map[string]string{}
	for k, v := range ENV {
		if v != "" {
			env[k] = v } }
	text_parts := strings.Split(text, SPAN_MARKER)
	text_left := strings.TrimSpace(text_parts[0])
	text_right := ""
	if len(text_parts) > 1 {
		text_right = strings.TrimSpace(text_parts[1]) }
	state := map[string]string {
		"SELECTED": selected,
		"SELECTION": strings.Join(SELECTION, "\n"),
		// either SELECTION or current row...
		"ACTIVE": strings.Join(getActiveList(), "\n"),
		// XXX need a way to tell the command the current available width...
		//"COLS": fmt.Sprint(CONTENT_COLS),
		//"ROWS": fmt.Sprint(CONTENT_ROWS),
		"LINES": fmt.Sprint(len(TEXT_BUFFER.Lines)),
		"LINE": fmt.Sprint(ROW_OFFSET + CURRENT_ROW + 1),
		"INDEX": fmt.Sprint(ROW_OFFSET + CURRENT_ROW),
		"TEXT": text,
		"TEXT_LEFT": text_left, 
		"TEXT_RIGHT": text_right, 
	}
	for k, v := range state {
		if v != "" {
			env[k] = v } }
	return env }
func makeCallEnv(cmd *exec.Cmd) []string {
	env := []string{}
	for k, v := range makeEnv() {
		env = append(env, k +"="+ v) }
	return append(cmd.Environ(), env...) }


// XXX need to rethink the external call API...
type Command struct {
	State string
	Done chan bool
	Kill chan bool
	Stdout *io.ReadCloser
	Stderr *io.ReadCloser
	Error error
}
// XXX should we use bytes.Buffer or cmd.StdoutPipe()/cmd.StderrPipe() ???
// XXX should stdin be io.ReadCloser???
func goCallCommand(code string, stdin io.Reader) Command {
	// build the command...
	shell := strings.Fields(SHELL)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)
	env := makeCallEnv(cmd)
	// io...
	cmd.Env = env
	// XXX can we make these optional???
	cmd.Stdin = stdin
	stdout, _ := cmd.StdoutPipe()
	// XXX for some reason we're not getting the error from the pipe...
	//stderr, _ := cmd.StderrPipe()
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	// output package...
	done := make(chan bool)
	kill := make(chan bool)
	res := Command {
		State: "pending",
		Done: done,
		Kill: kill,
		Stdout: &stdout, 
		//Stderr: &stderr,
	} 
	//SPINNER.Start()

	// handle killing the process when needed...
	watchdogDone := make(chan bool)
	go func(){
		//defer SPINNER.Stop()
		select {
			case <-kill:
				res.State = "killed"
				if err := cmd.Process.Kill() ; err != nil {
					log.Panic(err) } 
			case s := <-watchdogDone:
				if s == true {
					res.State = "done"
				} else {
					res.State = "failed" }
				return } }()

	// run...
	if err := cmd.Start(); err != nil {
		log.Panic(err) }

	// cleanup...
	go func(){
		done_state := true
		if err := cmd.Wait(); err != nil {
			log.Println("Error executing: \""+ code +"\"", err) 
			scanner := bufio.NewScanner(stderr)
			lines := []string{}
			for scanner.Scan() {
				lines = append(lines, scanner.Text()) }
			log.Println("    ERR:", strings.Join(lines, "\n"))
			log.Println("    ENV:", env)
			res.Error = err
			done_state = false }
		watchdogDone <- done_state
		done <- done_state }()

	return res }
func goCallTransform(code string, line string) Command {
	var stdin bytes.Buffer
	stdin.Write([]byte(line))
	return goCallCommand(code, &stdin) }


// XXX needs revision -- feels hacky...
// XXX use more generic input types -- io.Reader / io.Writer...
// XXX generalize and combine callAtCommand(..) and callCommand(..)
func callAtCommand(code string, stdin bytes.Buffer) error {
	shell := strings.Fields(SHELL)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)
	env := makeCallEnv(cmd)
	cmd.Env = env

	cmd.Stdin = &stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// NOTE: order here is significant...
	defer SCREEN.Sync()
	defer SCREEN.Resume()
	// XXX can we suspend but without flusing the screen???
	SCREEN.Suspend()

	// run the command...
	// XXX this should be run async???
	//		...option??
	var err error
	if err = cmd.Run(); err != nil {
		log.Println("Error executing: \""+ code +"\":", err) 
		log.Println("    ERR:", os.Stderr)
		log.Println("    ENV:", env) }

	return err }
func callCommand(code string, stdin bytes.Buffer) (bytes.Buffer, bytes.Buffer, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// XXX this hangs the app...
	//SPINNER.Start()
	//defer SPINNER.Stop()

	shell := strings.Fields(SHELL)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)
	env := makeCallEnv(cmd)
	cmd.Env = env

	// XXX can we make these optional???
	cmd.Stdin = &stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// run the command...
	var err error
	if err = cmd.Run(); err != nil {
		log.Println("Error executing: \""+ code +"\":", err) 
		log.Println("    ERR:", stderr.String())
		log.Println("    ENV:", env) }


	return stdout, stderr, err }
func callTransform(code string, line string) (string, error) {
	var stdin bytes.Buffer
	stdin.Write([]byte(line))
	stdout, _, err := callCommand(code, stdin)
	return stdout.String(), err }
var isVarCommand = regexp.MustCompile(`^\s*[a-zA-Z_]+=`)
//func goCallAction(actions string) Result {
func callAction(actions string) Result {
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
				stdin.Write([]byte(TEXT_BUFFER.Lines[CURRENT_ROW].text)) }

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
				//TEXT_BUFFER = LinesBuffer{}
				TEXT_BUFFER.Use.Lock()
				TEXT_BUFFER.Clear()
				for scanner.Scan() {
					/* XXX
					// update screen as soon as we reach selection and 
					// just after we fill the screen...
					if len(TEXT_BUFFER.Lines) == CURRENT_ROW || 
							len(TEXT_BUFFER.Lines) == CURRENT_ROW + ROW_OFFSET {
						TEXT_BUFFER.Use.Unlock() }
					//*/
					line := scanner.Text()
					lines = append(lines, line)
					TEXT_BUFFER.Push(line) } 
				if len(TEXT_BUFFER.Lines) == 0 {
					TEXT_BUFFER.Push("") } 
				TEXT_BUFFER.Use.Unlock()
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
			method := reflect.ValueOf(&ACTIONS).MethodByName(action)
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
				return value } } }
	return OK }
func callHandler(key string) Result {
	// expand aliases...
	seen := []string{ key }
	if action, exists := KEYBINDINGS[key] ; exists {
		_action := action
		for exists && ! slices.Contains(seen, _action) {
			if _action, exists = KEYBINDINGS[_action] ; exists {
				action = _action } }
		return callAction(action) }
	// call key alias...
	parts := strings.Split(key, "+")
	if aliases, exists := KEY_ALIASES[parts[len(parts)-1]] ; exists {
		for _, key := range aliases {
			res := callHandler(
				strings.Join(append(parts[:len(parts)-1], key), "+"))
			if res == Missing {
				log.Println("Key Unhandled:",
					strings.Join(append(parts[:len(parts)-1], key), "+"))
				continue }
			return res } }
	return Missing }

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
func evt2keys(evt tcell.EventKey) []string {
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
			Key = string(k)
			mods = append(mods, "shift") } 
		key = strings.ToLower(string(k)) } 

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

func handleScrollLimits(){
	delta := 0

	top_threshold := SCROLL_THRESHOLD_TOP
	bottom_threshold := ROWS - SCROLL_THRESHOLD_BOTTOM - 1 
	if ROWS < SCROLL_THRESHOLD_TOP + SCROLL_THRESHOLD_BOTTOM {
		top_threshold = ROWS / 2
		bottom_threshold = ROWS - top_threshold }
	
	// buffer smaller than screen -- keep at top...
	if ROWS > len(TEXT_BUFFER.Lines) {
		ROW_OFFSET = 0
		CURRENT_ROW -= ROW_OFFSET
		return }

	// keep from scrolling past the bottom of the screen...
	if ROW_OFFSET + ROWS > len(TEXT_BUFFER.Lines) {
		delta = ROW_OFFSET - (len(TEXT_BUFFER.Lines) - ROWS)
	// scroll to top threshold...
	} else if CURRENT_ROW < top_threshold && 
			ROW_OFFSET > 0 {
		delta = top_threshold - CURRENT_ROW
		if delta > ROW_OFFSET {
			delta = ROW_OFFSET }
	// keep current row on screen...
	} else if CURRENT_ROW > bottom_threshold && 
			CURRENT_ROW > top_threshold {
		delta = bottom_threshold - CURRENT_ROW
		// saturate delta...
		if delta < (ROW_OFFSET + ROWS) - len(TEXT_BUFFER.Lines) {
			delta = (ROW_OFFSET + ROWS) - len(TEXT_BUFFER.Lines) } } 

	// do the update...
	if delta != 0 {
		ROW_OFFSET -= delta 
		CURRENT_ROW += delta } }


func updateGeometry(screen tcell.Screen){
	err := error(nil)
	W, H := screen.Size()
	
	//WIDTH, HEIGHT = W, H

	// WIDTH...
	if SIZE[0] == "auto" {
		WIDTH = W
	} else if SIZE[0][len(SIZE[0])-1] == '%' {
		r, err := strconv.ParseFloat(string(SIZE[0][0:len(SIZE[0])-1]), 32)
		if err != nil {
			log.Println("Error parsing width", SIZE[0]) }
		WIDTH = int(float64(W) * (r / 100))
	} else {
		WIDTH, err = strconv.Atoi(SIZE[0])
		if err != nil {
			log.Println("Error parsing width", SIZE[0]) } }
	// HEIGHT...
	if SIZE[1] == "auto" {
		HEIGHT = H
	} else if SIZE[1][len(SIZE[1])-1] == '%' {
		r, err := strconv.ParseFloat(string(SIZE[1][0:len(SIZE[1])-1]), 32)
		if err != nil {
			log.Println("Error parsing height", SIZE[1]) }
		HEIGHT = int(float64(H) * (r / 100))
	} else {
		HEIGHT, err = strconv.Atoi(SIZE[1])
		if err != nil {
			log.Println("Error parsing height", SIZE[1]) } }
	// LEFT (value)
	left_set := false
	if slices.Contains(ALIGN, "left") {
		left_set = false
		LEFT = 0
	} else if slices.Contains(ALIGN, "right") {
		left_set = false
		LEFT = W - WIDTH
	} else if ALIGN[0] != "center" {
		left_set = false
		LEFT, err = strconv.Atoi(ALIGN[0])
		if err != nil {
			log.Println("Error parsing left", ALIGN[0]) } }
	// TOP (value)
	top_set := false
	if slices.Contains(ALIGN, "top") {
		top_set = false
		TOP = 0
	} else if slices.Contains(ALIGN, "bottom") {
		top_set = false
		TOP = W - WIDTH
	} else if ALIGN[1] != "center" {
		top_set = false
		TOP, err = strconv.Atoi(ALIGN[1]) 
		if err != nil {
			log.Println("Error parsing top", ALIGN[1]) } }
	// LEFT (center)
	if ! left_set {
		if top_set && 
				slices.Contains(ALIGN, "center") || 
				ALIGN[0] == "center" {
			LEFT = int(float64(W - WIDTH) / 2) } }
	// TOP (center)
	if ! top_set {
		if top_set && 
				slices.Contains(ALIGN, "center") || 
				ALIGN[0] == "center" {
			TOP = int(float64(H - HEIGHT) / 2) } }

	COLS, ROWS = WIDTH, HEIGHT
	if TITLE_LINE {
		ROWS-- }
	if STATUS_LINE {
		ROWS-- } }

var SCREEN tcell.Screen

// XXX RENAME -- this does init and event handling...
func lines() Result {
	// setup...
	screen, err := tcell.NewScreen()
	SCREEN = screen
	if err != nil {
		log.Fatalf("%+v", err) }
	if err := screen.Init(); err != nil {
		log.Fatalf("%+v", err) }
	screen.EnableMouse()
	screen.EnablePaste()

	// XXX add option to draw over terminal content (i.e. transparent bg)...
	// XXX need option to draw shadow...
	if t, ok := THEME["background"] ; ok {
		screen.SetStyle(t)
	} else if t, ok := THEME["default"] ; ok {
		screen.SetStyle(t)
	} else {
		screen.SetStyle(tcell.StyleDefault) }
	screen.Clear()

	// handle panics...
	quit := func() {
		maybePanic := recover()
		screen.Fini()
		if maybePanic != nil {
			panic(maybePanic) } }
	defer quit()

	// show empty screen...
	// XXX should also show spinner...
	updateGeometry(screen)
	drawScreen(screen, THEME)
	screen.Show()

	// load initial state...
	ACTIONS.Update()

	if SELECTION_CMD != "" {
		var stdin bytes.Buffer
		stdin.Write([]byte(TEXT_BUFFER.String()))
		stdout, _, err := callCommand(SELECTION_CMD, stdin)
		if err != nil {
			log.Println("Error executing:", SELECTION_CMD) 
		} else {
			SetSelection(strings.Split(stdout.String(), "\n")) } }

	for {
		updateGeometry(screen)
		drawScreen(screen, THEME)
		screen.Show()

		evt := screen.PollEvent()

		switch evt := evt.(type) {
			// keep the selection in the same spot...
			case *tcell.EventResize:
				updateGeometry(screen)
				handleScrollLimits()

			case *tcell.EventKey:
				for _, key := range evt2keys(*evt) {
					res := callHandler(key)
					if res == Missing {
						log.Println("Key Unhandled:", key)
						continue }
					if res != OK {
						return res } 
					break }
				//log.Println("KEY:", evt.Name())
				// defaults...
				if evt.Key() == tcell.KeyEscape || evt.Key() == tcell.KeyCtrlC {
					return OK }

			case *tcell.EventMouse:
				buttons := evt.Buttons()
				// get modifiers...
				// XXX this is almost the same as in evt2keys(..) can we generalize???
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
					col, row := evt.Position()
					// ignore clicks outside the list...
					if col < LEFT || col >= LEFT + WIDTH || 
							row < TOP || row >= TOP + HEIGHT {
						continue }
					// title/status bars and borders...
					top_offset := 0
					if TITLE_LINE {
						top_offset = 1
						if row == TOP {
							// XXX handle titlebar click???
							//log.Println("TITLE_LINE")
							continue } }
					if STATUS_LINE {
						if row - TOP == ROWS + 1 {
							// XXX handle statusbar click???
							//log.Println("STATUS_LINE")
							continue } }
					if BORDER > 0 {
						if col == LEFT ||
								(SCROLLBAR <= 0 && 
									col == LEFT + COLS - 1) {
							//log.Println("BORDER")
							continue } }
					// scrollbar...
					// XXX sould be nice if we started in the scrollbar 
					//		to keep handling the drag untill released...
					//		...for this to work need to either detect 
					//		drag or release...
					if SCROLLBAR > 0 && 
							col == LEFT + COLS - 1 {
						//log.Println("SCROLLBAR")
						ROW_OFFSET = 
							int((float64(row - TOP - top_offset) / float64(ROWS - 1)) * 
							float64(len(TEXT_BUFFER.Lines) - ROWS))
					// call click handler...
					} else {
						MOUSE_COL = col - LEFT - BORDER
						MOUSE_ROW = row - TOP
						if TITLE_LINE {
							MOUSE_ROW-- }

						// empty space below rows...
						if MOUSE_ROW >= len(TEXT_BUFFER.Lines) {
							if EMPTY_SPACE != "passive" {
								CURRENT_ROW = len(TEXT_BUFFER.Lines) - 1 }
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
							res := callHandler(key) 
							if res == Missing {
								continue }
							if res != OK {
								return res } 
							break } }
					handleScrollLimits()

				} else if buttons & tcell.WheelUp != 0 {
					// XXX add mods...
					res := callHandler("WheelUp")
					if res == Missing {
						res = OK }
					if res != OK {
						return res }

				} else if buttons & tcell.WheelDown != 0 {
					// XXX add mods...
					res := callHandler("WheelDown")
					if res == Missing {
						res = OK }
					if res != OK {
						return res } } } } }



// command line args...
var options struct {
	Pos struct {
		FILE string
	} `positional-args:"yes"`

	ListCommand string `short:"c" long:"cmd" value-name:"CMD" env:"CMD" description:"List command"`
	// NOTE: this is not the same as filtering the input as it will be 
	//		done lazily when the line reaches view.
	TransformCommand string `short:"t" long:"transform" value-name:"CMD" env:"TRANSFORM" description:"Row transform command"`
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
		ScrollThresholdTop int `long:"scroll-threshold-top" value-name:"N" default:"3" description:"Number of lines from the top edge of screen to triger scrolling"`
		ScrollThresholdBottom int `long:"scroll-threshold-bottom" value-name:"N" default:"3" description:"Number of lines from the bottom edge of screen to triger scrolling"`
		// XXX add named themes/presets...
		Theme map[string]string `long:"theme" value-name:"NAME:FGCOLOR:BGCOLOR" description:"Set theme color"`
	} `group:"Configuration"`

	Introspection struct {
		ListActions bool `long:"list-actions" description:"List available actions"`
		ListThemeable bool `long:"list-themeable" description:"List available themable element names"`
		ListBorderThemes bool `long:"list-border-themes" description:"List border theme names"`
		ListSpinners bool `long:"list-spinners" description:"List spinner styles"`
		ListColors bool `long:"list-colors" description:"List usable color names"`
	} `group:"Introspection"`
}


func startup() Result {
	parser := flags.NewParser(&options, flags.Default)

	_, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return OK }
		log.Println("Error:", err)
		os.Exit(1) }

	// doc...
	if options.Introspection.ListActions {
		t := reflect.TypeOf(&ACTIONS)
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			fmt.Println("    "+ m.Name) }
		return OK }
	if options.Introspection.ListThemeable {
		for name, _ := range THEME {
			fmt.Println("    "+ name) }
		return OK }
	if options.Introspection.ListBorderThemes {
		names := []string{}
		l := 0
		for name, _ := range BORDER_THEME {
			if len(name) > l {
				l = len(name) }
			names = append(names, name) }
		slices.Sort(names)
		for _, name := range names {
			fmt.Printf("    %-"+ fmt.Sprint(l) +"v \"%v\"\n", name, BORDER_THEME[name]) }
		return OK }
	if options.Introspection.ListSpinners {
		for i, style := range SPINNER_STYLES {
			fmt.Printf("    %3v \"%v\"\n", i, style) }
		return OK }
	if options.Introspection.ListColors {
		for name, _ := range tcell.ColorNames {
			fmt.Println("    "+ name) }
		return OK }

	// globals...
	INPUT_FILE = options.Pos.FILE
	LIST_CMD = options.ListCommand
	TRANSFORM_CMD = options.TransformCommand
	TRANSFORM_POPULATE_CMD = options.TransformPopulateCommand
	SELECTION_CMD = options.SelectionCommand

	// focus/positioning...
	FOCUS = options.Focus
	CURRENT_ROW = options.FocusRow
	FOCUS_CMD = options.FocusCmd

	if options.Chrome.Border ||  
			! parser.FindOptionByLongName("border-chars").IsSetDefault() {
		BORDER = 1 
		// char order: 
		//		 01234567
		//		"│┌─┐│└─┘"
		// XXX might be fun to add border themes...
		chars, ok := BORDER_THEME[options.Chrome.BorderChars]
		border_chars := []rune{}
		if ok {
			border_chars = []rune(chars)
		} else {
			border_chars = []rune(
				// normalize length...
				fmt.Sprintf("%-8v", options.Chrome.BorderChars)) }
		BORDER_LEFT = border_chars[0] 
		BORDER_RIGHT = border_chars[4] 
		BORDER_TOP = border_chars[2] 
		BORDER_BOTTOM = border_chars[6] 
		BORDER_CORNERS = map[string]rune{
			"ul": border_chars[1],	
			"ur": border_chars[3],	
			"ll": border_chars[5],	
			"lr": border_chars[7],	
		} }

	if i, err := strconv.Atoi(options.Chrome.SpinnerChars); err != nil {
		SPINNER.Frames = options.Chrome.SpinnerChars
	} else {
		SPINNER.Frames = SPINNER_STYLES[i] }

	TITLE_LINE_FMT = options.Chrome.Title
	TITLE_LINE = TITLE_LINE_FMT != ""
	TITLE_CMD = options.Chrome.TitleCommand

	STATUS_LINE_FMT = options.Chrome.Status
	STATUS_LINE = STATUS_LINE_FMT != ""
	STATUS_CMD = options.Chrome.StatusCommand

	SIZE = strings.Split(options.Chrome.Size, ",")
	ALIGN = strings.Split(options.Chrome.Align, ",")
	TAB_SIZE = options.Chrome.Tab
	SPAN_MODE = options.Chrome.Span
	//SPAN_MARKER = options.Chrome.SpanMarker
	SPAN_EXTEND = options.Chrome.SpanExtend
	SPAN_LEFT_MIN_WIDTH = options.Chrome.SpanLeftMin
	SPAN_RIGHT_MIN_WIDTH = options.Chrome.SpanRightMin
	SPAN_FILLER = []rune(options.Chrome.SpanFiller)[0]
	SPAN_FILLER_TITLE = []rune(fmt.Sprintf("%1v", options.Chrome.SpanFillerTitle))[0]
	SPAN_FILLER_STATUS = []rune(fmt.Sprintf("%1v", options.Chrome.SpanFillerStatus))[0]
	// defaults to SPAN_FILLER...
	SPAN_SEPARATOR = SPAN_FILLER
	if ! parser.FindOptionByLongName("span-separator").IsSetDefault() {
		SPAN_SEPARATOR = []rune(fmt.Sprintf("%1v", options.Chrome.SpanSeparator))[0] }
	OVERFLOW_INDICATOR = []rune(options.Chrome.OverflowIndicator)[0]
	EMPTY_SPACE = options.Chrome.EmptySpace
	// defaults to .ScrollThreshold...
	SCROLL_THRESHOLD_TOP = options.Config.ScrollThreshold
	if ! parser.FindOptionByLongName("scroll-threshold-top").IsSetDefault() {
		SCROLL_THRESHOLD_TOP = options.Config.ScrollThresholdTop }
	SCROLL_THRESHOLD_BOTTOM = options.Config.ScrollThreshold
	if ! parser.FindOptionByLongName("scroll-threshold-bottom").IsSetDefault() {
		SCROLL_THRESHOLD_BOTTOM = options.Config.ScrollThresholdBottom }
	
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

	// themes/colors...
	for name, spec := range options.Config.Theme {
		color := strings.SplitN(spec, ":", 2)
		THEME[name] = 
			tcell.StyleDefault.
				Foreground(tcell.GetColor(color[0])).
				Background(tcell.GetColor(color[1])) }

	// log...
	logFileName := options.Config.LogFile
	// XXX can we suppress only log.Print*(..) and keep errors and panic output???
	if logFileName == "" {
		logFileName = "/dev/null" }
	logFile, err := os.OpenFile(
		logFileName,
		os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err) }
	defer logFile.Close()
	// Set log out put and enjoy :)
	log.SetOutput(logFile) 

	// startup...
	res := lines() 
	if res == Fail {
		return res }

	// output...
	// XXX should this be here or in lines(..)
	if STDOUT != "" {
		fmt.Println(STDOUT) } 
	return OK }


func main(){
	os.Exit(toExitCode(startup())) }


// vim:set sw=4 ts=4 nowrap :
