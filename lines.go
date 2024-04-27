/*
*
* Features:
*	- list with line navigation
*	- selection
*	- actions
*	- live search/filtering
*
*
* XXX BUG: scrollbar sometimes is off by 1 cell when scrolling down (small overflow)...
*
*
* XXX handle paste (and copy) -- actions...
* XXX can we run two instances and tee input/output???
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
*
*/

package main

import "os"
import "os/exec"
//import "io"
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

	ListCMD string
	TransformCMD string
	InputFile string
	// XXX should this be bytes.Buffer???
	Output string

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
}

func New() Lines {
	return Lines{
		TabSize: 8,
		Shell: "bash -c",
		Theme: THEME,
		Keybindings: KEYBINDINGS,
		// XXX
	}
}
//*/



var LIST_CMD string
var TRANSFORM_CMD string
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

var COL_OFFSET = 0
var ROW_OFFSET = 0

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

// XXX should this be '|' ???
var SPAN_MARKER = "%SPAN"
//var SPAN_MODE = "fit-right"
var SPAN_MODE = "10"
var SPAN_LEFT_MIN_WIDTH = 8
var SPAN_RIGHT_MIN_WIDTH = 8
//var SPAN_SEPARATOR = tcell.RuneVLine
var SPAN_SEPARATOR = ' '

var OVERFLOW_INDICATOR = '}'

// current row relative to viewport...
var CURRENT_ROW = 0

// XXX cursor mode...
//		- cursor
//		- line
//		- page
//		- pattern


var TEXT_BUFFER_WIDTH = 0

type Row struct {
	selected bool
	transformed bool
	text string
}
var TEXT_BUFFER = []Row{}

var SELECTION = []string{}

// XXX load this from config...
// XXX how do we represent other keys???
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

	"Insert": `
		SelectToggle
		Down`,
	"Space": `
		SelectToggle
		Down`,
	"ctrl+a": "SelectAll",
	// XXX ctrl-i is Tab -- can we destinguish the two in the terminal???
	"ctrl+i": "SelectInverse",
	"ctrl+d": "SelectNone",

	"ctrl+r": "Refresh",
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
	"status-line": tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorReset),
	"title-line": tcell.StyleDefault.
		Background(tcell.ColorGray).
		Foreground(tcell.ColorReset),
	"background": tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset),
	//"hover": tcell.StyleDefault.
	//	Background(tcell.ColorGray).
	//	Foreground(tcell.ColorReset),
}


// XXX these are almost identical, can we generalize?
// XXX option to maintain current row...
func str2buffer(str string){
	CURRENT_ROW = 0
	TEXT_BUFFER = []Row{}
	n := 0
	for _, str := range strings.Split(str, "\n") {
		row := Row{ text: str }
		TEXT_BUFFER = append(TEXT_BUFFER, row)
		// set max line width...
		l := len([]rune(row.text))
		if TEXT_BUFFER_WIDTH < l {
			TEXT_BUFFER_WIDTH = l }
		n++ }
	// keep at least one empty line in buffer...
	// XXX should we do this here or in the looping code???
	if n == 0 {
		TEXT_BUFFER = append(TEXT_BUFFER, Row{}) } }
func file2buffer(file *os.File){
	// XXX set this to a logical value...
	CURRENT_ROW = 0
	TEXT_BUFFER = []Row{}
	n := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan(){
		row := Row{ text: scanner.Text() }
		TEXT_BUFFER = append(TEXT_BUFFER, row)
		// set max line width...
		l := len([]rune(row.text))
		if TEXT_BUFFER_WIDTH < l {
			TEXT_BUFFER_WIDTH = l }
		n++ }
	// keep at least one empty line in buffer...
	// XXX should we do this here or in the looping code???
	if n == 0 {
		TEXT_BUFFER = append(TEXT_BUFFER, Row{}) } }



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
	size := false
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
			if len(TEXT_BUFFER) != 0 {
				current = TEXT_BUFFER[CURRENT_ROW + ROW_OFFSET] } 
			switch name {
				// this has to be handled later, when the string is 
				// otherwise complete...
				case string(SPAN_MARKER[1:]):
					size = true
					val = SPAN_MARKER
				case "CMD":
					if cmd != "" {
						val, err = callTransform(cmd, str)
						if err != nil {
							val = "" } }
				case "SELECTED":
					val = ""
					if current.selected {
						val = "*" }
				case "SELECTION":
					val = fmt.Sprint(len(SELECTION))
					if val == "0" {
						val = "" }
				case "REST":
					val = current.text[COLS:] }
			return []byte(val) }))

	// %SPAN / fit width...
	if size {
		// contract...
		if len(str) > COLS {
			overflow := (len(str) - COLS) + 3
			parts := strings.SplitN(str, SPAN_MARKER, 2)
			str = string(parts[0][:len(parts[0])-overflow]) + "..."
			if len(parts) > 1 {
				str += parts[1] }
		// expand...
		} else {
			// XXX need a way to indicate the character to use for expansion...
			str = strings.Replace(str, SPAN_MARKER, 
				fmt.Sprintf("%"+fmt.Sprint(COLS - len(str) + len(SPAN_MARKER)) +"v", ""), 1) } }
	return str }

func drawScreen(screen tcell.Screen, theme Theme){
	screen.Clear()

	// scrollbar...
	var scroller_size, scroller_offset int
	scroller_style, ok := theme["scrollbar"]
	if ! ok {
		scroller_style = theme["default"] }
	if len(TEXT_BUFFER) > ROWS {
		SCROLLBAR = 1
	} else {
		SCROLLBAR = 0 }
	if SCROLLBAR > 0 {
		r := float64(ROWS) / float64(len(TEXT_BUFFER))
		scroller_size = 1 + int(float64(ROWS - 1) * r)
		scroller_offset = int(float64(ROW_OFFSET + 1) * r) }

	// XXX CONTENT_ROWS... (???)
	top_offset := 0
	bottom_offset := 0
	if TITLE_LINE {
		top_offset++ }
	if STATUS_LINE {
		bottom_offset++ }
	height := 
		top_offset + 
		ROWS + 
		bottom_offset

	var col, row int
	style := theme["default"]
	for row = TOP ; row < TOP + height ; row++ {
		var buf_row = row - top_offset + ROW_OFFSET - TOP

		// row theming...
		style = theme["default"]
		non_default_style := true
		if buf_row >= 0 && 
				buf_row < len(TEXT_BUFFER) {
			// current+selected...
			if TEXT_BUFFER[buf_row].selected &&
					CURRENT_ROW == row - top_offset - TOP {
				style, non_default_style = theme["current-selected"]
			// mark selected...
			} else if TEXT_BUFFER[buf_row].selected {
				style, non_default_style = theme["selected"]
			// hover...
			// XXX do we need this???
			//} else if HOVER_ROW == row - top_offset - TOP {
			//	style, non_default_style = theme["hover"] 
			// current...
			} else if CURRENT_ROW == row - top_offset - TOP {
				style, non_default_style = theme["current"] } } 

		// normalize...
		line := []rune{}
		// buffer line...
		if buf_row >= 0 && 
				buf_row < len(TEXT_BUFFER) && 
				row >= TOP + top_offset &&
				row <= TOP + ROWS {
			// transform (lazy)...
			// XXX should we do this in advance +/- screen (a-la ImageGrid ribbons)???
			if TRANSFORM_CMD != "" && 
					! TEXT_BUFFER[buf_row].transformed {
				text, err := callTransform(TRANSFORM_CMD, TEXT_BUFFER[buf_row].text)
				if err == nil {
					TEXT_BUFFER[buf_row].text = text
					TEXT_BUFFER[buf_row].transformed = true } }
			line = []rune(TEXT_BUFFER[buf_row].text) 
		// chrome...
		} else {
			str, cmd := "", ""
			// title...
			if TITLE_LINE && 
					row == TOP {
				str = TITLE_LINE_FMT
				style, non_default_style = theme["title-line"]
				if TITLE_CMD != "" {
					cmd = TITLE_CMD }
			// status...
			} else if STATUS_LINE && 
					row == TOP + height-1 {
				str = STATUS_LINE_FMT
				style, non_default_style = theme["status-line"] 
				if STATUS_CMD != "" {
					cmd = STATUS_CMD } }
			// populate the line...
			line = []rune(populateTemplateLine(str, cmd)) }

		// set default style...
		if ! non_default_style {
			style = theme["default"] }

		// draw row...
		var col_offset = 0
		var buf_offset = 0
		for col = LEFT ; col < LEFT + COLS - col_offset ; col++ {
			cur_col := col + col_offset
			buf_col := col + buf_offset + COL_OFFSET - LEFT

			// content block...
			content_block := false
			if ! (TITLE_LINE &&
						row < TOP + top_offset) &&
					! (STATUS_LINE &&
						row == TOP + height-1) {
				content_block = true }

			// scrollbar...
			if content_block &&
					SCROLLBAR > 0 && 
					cur_col == LEFT + COLS-1 {
				c := SCROLLBAR_BG
				if row - top_offset - TOP >= scroller_offset && 
						row - top_offset - TOP < scroller_offset + scroller_size {
					c = SCROLLBAR_FG }
				screen.SetContent(cur_col, row, c, nil, scroller_style)
				continue }

			// get rune...
			c := ' '
			if buf_col < len(line) {
				c = line[buf_col] } 

			// escape sequences...
			// see: 
			//	https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797 
			// XXX need to handle colors at least...
			if c == '\x1B' {
				for c == '\x1B' {
					i := buf_col + 1
					if line[i] == '[' {
						ansi_commands := "HfABCDEFGnsuJKmhlp"
						for i < len(line) && 
								! strings.ContainsRune(ansi_commands, line[i]) {
							i++ }
						/*/ XXX handle color...
						if line[i] == 'm' {
							style = ansi2style(string(line[buf_col:i+1]), style) }
						//*/
					} else {
						ansi_direct_commands := "M78"
						for i < len(line) && 
								! strings.ContainsRune(ansi_direct_commands, line[i]) {
							i++ } } 
					buf_offset += (i + 1) - buf_col
					buf_col = i + 1
					if buf_col >= len(line) {
						c = ' ' 
					} else {
						c = line[buf_col] } }
				// pass the next rune throu the whole stack...
				col--
				continue }

			// overflow indicator...
			if content_block &&
					buf_col + col_offset == COLS - SCROLLBAR - 1 && 
					buf_col < len(line)-1 {
				screen.SetContent(cur_col, row, OVERFLOW_INDICATOR, nil, style)
				continue } 

			// "%SPAN" -- expand/contract line to fit width...
			if c == '%' && 
					string(line[buf_col:buf_col+len(SPAN_MARKER)]) == SPAN_MARKER {
				offset := 0
				// automatic -- align to left/right edges...
				// NOTE: this essentially rigth-aligns the right side, it 
				//		will not attempt to left-align to the SPAN_SEPARATOR...
				// XXX should we attempty to draw a sraight vertical line between columns???
				if SPAN_MODE == "fit-right" {
					if len(line) - buf_col + SPAN_LEFT_MIN_WIDTH < COLS {
						offset = COLS - len(line) - SCROLLBAR
					} else {
						offset = -buf_col + SPAN_LEFT_MIN_WIDTH - SCROLLBAR }
				// manual...
				} else {
					c := 0
					// %...
					if SPAN_MODE[len(SPAN_MODE)-1] == '%' {
						p, err := strconv.ParseFloat(string(SPAN_MODE[0:len(SPAN_MODE)-1]), 64)
						if err != nil {
							log.Println("Error parsing:", SPAN_MODE) }
						c = int(float64(COLS) * (p / 100))
						// normalize...
						if c < SPAN_LEFT_MIN_WIDTH {
							c = SPAN_LEFT_MIN_WIDTH }
						if COLS - c < SPAN_RIGHT_MIN_WIDTH {
							c = COLS - SPAN_RIGHT_MIN_WIDTH }
						if COLS < SPAN_LEFT_MIN_WIDTH + SPAN_RIGHT_MIN_WIDTH {
							r := float64(SPAN_LEFT_MIN_WIDTH) / float64(SPAN_RIGHT_MIN_WIDTH) 
							c = int(float64(COLS) * r) }
					// cols...
					} else {
						v, err := strconv.Atoi(SPAN_MODE) 
						if err != nil {
							log.Println("Error parsing:", SPAN_MODE) 
							continue }
						if v < 0 {
							c = COLS + v - SCROLLBAR
						} else {
							c = v } }
					offset = c - buf_col - len(SPAN_MARKER) }
				i := cur_col
				for ; i < cur_col + offset + len(SPAN_MARKER) - 1 && i < LEFT + COLS ; i++ {
					screen.SetContent(i, row, ' ', nil, style) } 
				// separator...
				if col + offset + len(SPAN_MARKER) < LEFT + COLS { 
					sep := SPAN_SEPARATOR
					if offset - col_offset + len(SPAN_MARKER) - 1 < 0 {
						sep = OVERFLOW_INDICATOR }
					screen.SetContent(col + offset + len(SPAN_MARKER) - 1, row, sep, nil, style) 
				} else {
					screen.SetContent(LEFT + COLS - SCROLLBAR - 1, row, OVERFLOW_INDICATOR, nil, style) }
				col_offset = offset
				// skip the marker...
				col += len(SPAN_MARKER) - 1
				continue }

			// tab -- offset output to next tabstop... 
			if c == '\t' {
				// NOTE: the -1 here is to compensate fot the removed '\t'...
				offset := TAB_SIZE - ((buf_col + col_offset) % TAB_SIZE) - 1
				i := 0
				for ; i <= offset && cur_col + i < LEFT + COLS ; i++ {
					screen.SetContent(cur_col + i, row, ' ', nil, style) }
				// overflow indicator...
				if cur_col + i >= LEFT + COLS {
					screen.SetContent(cur_col + i - 1, row, OVERFLOW_INDICATOR, nil, style) }
				col_offset += offset 
				continue }

			// normal characters...
			screen.SetContent(cur_col, row, c, nil, style) } } }


// Actions...
// XXX since termbox is global, is there a point in holding any local 
//		data here???
// XXX can this be a map???
type Actions struct {}

type Result int
const (
	// Normal action return value.
	OK Result = -1 + iota

	// Returning this from an action will quit lines (exit code 0)
	Exit 

	// Returning this will quite lines with error (exit code 1)
	Fail
)
// Convert from Result type to propper exit code.
func toExitCode(r Result) int {
	if r == Fail {
		return int(Fail) }
	return 0 }

// vertical navigation...
// XXX changing only CURRENT_ROW can be donwe by redrawing only two lines...
func (this Actions) Up() Result {
	if CURRENT_ROW > 0 && 
			// account for SCROLL_THRESHOLD_TOP...
			(CURRENT_ROW > SCROLL_THRESHOLD_TOP ||
				ROW_OFFSET == 0) {
		CURRENT_ROW-- 
	// scroll the buffer...
	} else {
		this.ScrollUp() }
	return OK }
func (this Actions) Down() Result {
	// within the text buffer...
	if CURRENT_ROW + ROW_OFFSET < len(TEXT_BUFFER)-1 && 
			// within screen...
			CURRENT_ROW < ROWS-1 && 
			// buffer smaller than screen...
			(ROWS >= len(TEXT_BUFFER) ||
				// screen at end of buffer...
				ROW_OFFSET + ROWS == len(TEXT_BUFFER) ||
				// at scroll threshold...
				CURRENT_ROW < ROWS - SCROLL_THRESHOLD_BOTTOM - 1) {
		CURRENT_ROW++ 
	// scroll the buffer...
	} else {
		this.ScrollDown() }
	return OK }

// XXX should these track CURRENT_ROW relative to screen (current) or 
//		relative to content???
func (this Actions) ScrollUp() Result {
	if ROW_OFFSET > 0 {
		ROW_OFFSET-- }
	return OK }
func (this Actions) ScrollDown() Result {
	if ROW_OFFSET + ROWS < len(TEXT_BUFFER) {
		ROW_OFFSET++ } 
	return OK }

func (this Actions) PageUp() Result {
	if ROW_OFFSET > 0 {
		ROW_OFFSET -= ROWS 
		if ROW_OFFSET < 0 {
			this.Top() } 
	} else if ROW_OFFSET == 0 {
		this.Top() } 
	return OK }
func (this Actions) PageDown() Result {
	if len(TEXT_BUFFER) < ROWS {
		CURRENT_ROW = len(TEXT_BUFFER) - 1
		return OK }
	offset := len(TEXT_BUFFER) - ROWS
	if ROW_OFFSET < offset {
		ROW_OFFSET += ROWS 
		if ROW_OFFSET > offset {
			this.Bottom() } 
	} else if ROW_OFFSET == offset {
		this.Bottom() } 
	return OK }

func (this Actions) Top() Result {
	if ROW_OFFSET == 0 {
		CURRENT_ROW = 0 
	} else {
		ROW_OFFSET = 0 }
	return OK }
func (this Actions) Bottom() Result {
	if len(TEXT_BUFFER) < ROWS {
		CURRENT_ROW = len(TEXT_BUFFER) - 1
		return OK }
	offset := len(TEXT_BUFFER) - ROWS 
	if ROW_OFFSET == offset {
		CURRENT_ROW = ROWS - 1
	} else {
		ROW_OFFSET = len(TEXT_BUFFER) - ROWS }
	return OK }

/*// XXX horizontal navigation...
func (this Actions) Left() Result {
	// XXX
	return OK }
func (this Actions) Right() Result {
	// XXX
	return OK }

func (this Actions) ScrollLeft() Result {
	// XXX
	return OK }
func (this Actions) ScrollRight() Result {
	// XXX
	return OK }

func (this Actions) LeftEdge() Result {
	// XXX
	return OK }
func (this Actions) RightEdge() Result {
	// XXX
	return OK }
//*/

// selection...
func updateSelectionList(){
	SELECTION = []string{}
	for _, row := range TEXT_BUFFER {
		if row.selected {
			SELECTION = append(SELECTION, row.text) } } }
func (this Actions) SelectToggle(rows ...int) Result {
	if len(rows) == 0 {
		rows = []int{CURRENT_ROW + ROW_OFFSET} }
	for _, i := range rows {
		if TEXT_BUFFER[i].selected {
			TEXT_BUFFER[i].selected = false 
		} else {
			TEXT_BUFFER[i].selected = true } }
	updateSelectionList()
	return OK }
func (this Actions) SelectAll() Result {
	for i := 0; i < len(TEXT_BUFFER); i++ {
		TEXT_BUFFER[i].selected = true } 
	updateSelectionList()
	return OK }
func (this Actions) SelectNone() Result {
	for i := 0; i < len(TEXT_BUFFER); i++ {
		TEXT_BUFFER[i].selected = false } 
	SELECTION = []string{}
	return OK }
func (this Actions) SelectInverse() Result {
	rows := []int{}
	for i := 0 ; i < len(TEXT_BUFFER) ; i++ {
		rows = append(rows, i) }
	return this.SelectToggle(rows...) }

// utility...
func (this Actions) Update() Result {
	// file...
	if INPUT_FILE != "" {
		file, err := os.Open(INPUT_FILE)
		if err != nil {
			fmt.Println(err)
			return Fail }
		defer file.Close()
		file2buffer(file) 
	// command...
	} else if LIST_CMD != "" {
		return callAction("<"+ LIST_CMD)
	// pipe...
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalf("%+v", err) }
		if stat.Mode() & os.ModeNamedPipe != 0 {
			// XXX do we need to close this??
			//defer os.Stdin.Close()
			file2buffer(os.Stdin) } }
	return OK }
func (this Actions) Refresh() Result {
	SCREEN.Sync()
	return OK }

func (this Actions) Fail() Result {
	return Fail }
func (this Actions) Exit() Result {
	return Exit }

var ACTIONS Actions


var ENV = map[string]string {}
func makeEnv() map[string]string {
	// pass data to command via env...
	selected := ""
	text := ""
	// vars we need text for...
	if len(TEXT_BUFFER) > 0 { 
		if TEXT_BUFFER[CURRENT_ROW + ROW_OFFSET].selected {
			selected = "*" }
		text = TEXT_BUFFER[CURRENT_ROW + ROW_OFFSET].text }
	env := map[string]string{}
	for k, v := range ENV {
		if v != "" {
			env[k] = v } }
	state := map[string]string {
		"SELECTED": selected,
		"SELECTION": strings.Join(SELECTION, "\n"),
		// XXX need a way to tell the command the current available width...
		//"COLS": fmt.Sprint(CONTENT_COLS),
		//"ROWS": fmt.Sprint(CONTENT_ROWS),
		"LINES": fmt.Sprint(len(TEXT_BUFFER)),
		"LINE": fmt.Sprint(ROW_OFFSET + CURRENT_ROW + 1),
		"INDEX": fmt.Sprint(ROW_OFFSET + CURRENT_ROW),
		"TEXT": text,
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

// XXX needs revision -- feels hacky...
// XXX use more generic input types -- io.Reader / io.Writer...
// XXX generalize and combine callAtCommand(..) and callCommand(..)
func callAtCommand(code string, stdin bytes.Buffer) error {
	shell := strings.Fields(SHELL)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)

	cmd.Stdin = &stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := makeCallEnv(cmd)
	cmd.Env = env

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
		log.Println("    ENV:", env) }

	return err }
func callCommand(code string, stdin bytes.Buffer) (bytes.Buffer, bytes.Buffer, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	shell := strings.Fields(SHELL)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)

	// XXX can we make these optional???
	cmd.Stdin = &stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	env := makeCallEnv(cmd)
	cmd.Env = env

	//defer SCREEN.Sync()

	// run the command...
	// XXX this should be run async???
	//		...option??
	var err error
	if err = cmd.Run(); err != nil {
		log.Println("Error executing: \""+ code +"\":", err) 
		log.Println("    ENV:", env) }

	return stdout, stderr, err }

func callTransform(cmd string, line string) (string, error) {
	var stdin bytes.Buffer
	stdin.Write([]byte(line))
	stdout, _, err := callCommand(cmd, stdin)
	return stdout.String(), err }
var isVarCommand = regexp.MustCompile(`^\s*[a-zA-Z_]+=`)
// XXX add support for async commands...
func callAction(actions string) Result {
	// XXX make split here a bit more cleaver:
	//		- support ";"
	//		- support quoting of separators, i.e. ".. \\\n .." and ".. \; .."
	//		- ignore string literal content...
	for _, action := range strings.Split(actions, "\n") {
		//action = strings.Trim(action, " \t")
		action = strings.TrimSpace(action)
		if len(action) == 0 {
			continue }

		// NAME=ACTION...
		name := ""
		if isVarCommand.Match([]byte(action)) {
			parts := regexp.MustCompile("=").Split(action, 2)
			name, action = parts[0], parts[1] }
		// empty value -> remove from env...
		if name != "" && action == "" {
			delete(ENV, name) 
			continue }

		// shell commands:
		//		@ CMD	- simple/interactive command
		//					NOTE: this uses os.Stdout...
		//		! CMD	- stdout treated as env variables, one per line
		//		< CMD	- stdout read into buffer
		//		> CMD	- stdout printed to lines stdout
		//		| CMD	- current line passed to stdin
		//		XXX & CMD	- async command (XXX not implemented...)
		// NOTE: commands can be combined.
		prefixes := "@!<>|"
		prefix := []rune{}
		code := action
		// split out the prefixes...
		for strings.ContainsRune(prefixes, rune(code[0])) {
			prefix = append(prefix, rune(code[0]))
			code = strings.TrimSpace(string(code[1:])) }
		if len(prefix) > 0 {

			var stdin bytes.Buffer
			if slices.Contains(prefix, '|') {
				stdin.Write([]byte(TEXT_BUFFER[CURRENT_ROW].text)) }

			// call the command...
			var err error
			var output string
			if slices.Contains(prefix, '@') {
				err = callAtCommand(code, stdin)
			} else {
				var stdout bytes.Buffer
				stdout, _, err = callCommand(code, stdin)
				output = stdout.String() }
			if err != nil {
				log.Println("Error:", err)
				return Fail }

			// list output...
			// XXX stdout should be read line by line as it comes...
			// XXX keep selection and current item and screen position 
			//		relative to current..
			if slices.Contains(prefix, '<') {
				// ignore trailing \n's...
				for output[len(output)-1] == '\n' && len(output) > 0 {
					output = string(output[:len(output)-1]) }
				str2buffer(output) }
			// output to stdout...
			if slices.Contains(prefix, '>') {
				STDOUT += output }
			// output to env...
			if slices.Contains(prefix, '!') {
				for _, str := range strings.Split(output, "\n") {
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
					STDOUT += output
				} else {
					ENV[name] = output } }

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
	return OK }


// XXX modifier building in is not done yet...
func evt2keys(evt tcell.EventKey) []string {
	key_seq := []string{}
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

	// XXX STUB -- still need 3 and 4 mod combinations for completeness...
	//		...generate combinations + sort by length...
	for i := 0; i < len(mods); i++ {
		for j := i+1; j < len(mods); j++ {
			key_seq = append(key_seq, mods[i] +"+"+ mods[j] +"+"+ key) } }
	for _, m := range mods {
		key_seq = append(key_seq, m +"+"+ key) }
	// uppercase letter...
	if shifted {
		key_seq = append(key_seq, Key) }
	key_seq = append(key_seq, key)

	//log.Println("KEYS:", key, mods, key_seq)

	return key_seq }

func handleScrollLimits(){
	delta := 0

	top_threshold := SCROLL_THRESHOLD_TOP
	bottom_threshold := ROWS - SCROLL_THRESHOLD_BOTTOM - 1 
	if ROWS < SCROLL_THRESHOLD_TOP + SCROLL_THRESHOLD_BOTTOM {
		top_threshold = ROWS / 2
		bottom_threshold = ROWS - top_threshold }
	
	// buffer smaller than screen -- keep at top...
	if ROWS > len(TEXT_BUFFER) {
		ROW_OFFSET = 0
		CURRENT_ROW -= ROW_OFFSET
		return }

	// keep from scrolling past the bottom of the screen...
	if ROW_OFFSET + ROWS > len(TEXT_BUFFER) {
		delta = ROW_OFFSET - (len(TEXT_BUFFER) - ROWS)
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
		if delta < (ROW_OFFSET + ROWS) - len(TEXT_BUFFER) {
			delta = (ROW_OFFSET + ROWS) - len(TEXT_BUFFER) } } 

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

	// XXX should this be done in the event loop???
	ACTIONS.Update()

	for {
		updateGeometry(screen)

		/* XXX these are not used...
		CONTENT_COLS, CONTENT_ROWS = COLS, ROWS
		if SCROLLBAR > 0 {
			CONTENT_COLS-- }
		if TITLE_LINE {
			CONTENT_ROWS-- }
		if STATUS_LINE {
			CONTENT_ROWS-- }
		//*/

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
					if res != OK {
						return res } }
				//log.Println("KEY:", evt.Name())
				// defaults...
				if evt.Key() == tcell.KeyEscape || evt.Key() == tcell.KeyCtrlC {
					return OK }

			case *tcell.EventMouse:
				buttons := evt.Buttons()
				// XXX handle double click...
				// XXX handle modifiers...
				if buttons & tcell.Button1 != 0 || buttons & tcell.Button2 != 0 {
					col, row := evt.Position()
					//HOVER_COL, HOVER_ROW = col, row
					// ignore clicks outside the list...
					if col < LEFT || col >= LEFT + WIDTH || 
							row < TOP || row >= TOP + HEIGHT {
						continue }
					// title/status bars...
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
							float64(len(TEXT_BUFFER) - ROWS))
					// second click in curent row...
					// XXX should we have a timeout here???
					// XXX this triggers on drag... is this a bug???
					} else if row - top_offset - TOP == CURRENT_ROW {
						res := callHandler("ClickSelected") 
						if res != OK {
							return res }
					// below list...
					} else if row - TOP > len(TEXT_BUFFER) {
						CURRENT_ROW = len(TEXT_BUFFER) - 1
					// list...
					} else {
						CURRENT_ROW = row - TOP - top_offset}
					handleScrollLimits()


				} else if buttons & tcell.WheelUp != 0 {
					res := callHandler("WheelUp")
					if res != OK {
						return res }

				} else if buttons & tcell.WheelDown != 0 {
					res := callHandler("WheelDown")
					if res != OK {
						return res } } } } }



// command line args...
var options struct {
	Pos struct {
		FILE string
	} `positional-args:"yes"`

	// XXX can we set default values from variables???
	//		...doing ` ... `+ VAR +` ... ` breaks things...
	// XXX formatting the config string seems to break things...
	//ListCommand string `
	//	short:"c" 
	//	long:"cmd" 
	//	value-name:"CMD" 
	//	env:"CMD" 
	//	description:"List command"`
	// NOTE: this is not the same as filtering the input as it will be 
	ListCommand string `short:"c" long:"cmd" value-name:"CMD" env:"CMD" description:"List command"`
	// NOTE: this is not the same as filtering the input as it will be 
	//		done lazily when the line reaches view.
	TransformCommand string `short:"t" long:"transform" value-name:"CMD" env:"TRANSFORM" description:"Row transform command"`

	// XXX chicken-egg: need to first parse the args then parse the ini 
	//		and then merge the two...
	//ArgsFile string `long:"args-file" value-name:"FILE" env:"ARGS" description:"Arguments file"`


	// Quick actions...
	Actions struct {
		Select string `short:"s" long:"select" value-name:"ACTION" env:"SELECT" description:"Command to execute on item select"`
		Reject string `short:"r" long:"reject" value-name:"ACTION" env:"REJECT" description:"Command to execute on reject"`
	} `group:"Actions"`

	Keyboard struct {
		Key map[string]string `short:"k" long:"key" value-name:"KEY:ACTION" description:"Bind key to action"`
	} `group:"Keyboard"`

	Chrome struct {
		Title string `long:"title" value-name:"FMT" env:"TITLE" default:" %CMD " description:"Title format"`
		TitleCommand string `long:"title-cmd" value-name:"CMD" env:"TITLE_CMD" description:"Title command"`
		Status string `long:"status" value-name:"FMT" env:"STATUS" default:" %CMD %SPAN $LINE/$LINES " description:"Status format"`
		StatusCommand string `long:"status-cmd" value-name:"CMD" env:"STATUS_CMD" description:"Status command"`
		Size string `long:"size" value-name:"WIDTH,HEIGHT" env:"SIZE" default:"auto,auto" description:"Widget size"`
		Align string `long:"align" value-name:"LEFT,TOP" env:"ALIGN" default:"center,center" description:"Widget alignment"`
		Span string `long:"span" value-name:"MODE" env:"ALIGN" default:"fit-right" description:"Line spanning mode/size"`
		SpanSeparator string `long:"span-separator" value-name:"CHR" env:"SPAN_SEPARATOR" default:" " description:"Span separator character"`
		SpanLeftMin int `long:"span-left-min" value-name:"COLS" env:"SPAN_LEFT_MIN" default:"8" description:"Left column minimum span"`
		SpanRightMin int `long:"span-right-min" value-name:"COLS" env:"SPAN_RIGHT_MIN" default:"6" description:"Right column minimum span"`
		OverflowIndicator string `long:"overflow-indicator" value-name:"CHR" env:"OVERFLOW_INDICATOR" default:"}" description:"Line overflow character"`
		Tab int `long:"tab" value-name:"COLS" env:"TABSIZE" default:"8" description:"Tab size"`
	} `group:"Chrome"`

	Config struct {
		LogFile string `short:"l" long:"log" value-name:"FILE" env:"LOG" description:"Log file"`
		Separator string `long:"separator" value-name:"STRING" default:"\\n" env:"SEPARATOR" description:"Command separator"`
		// XXX might be fun to be able to set this to something like "middle"...
		ScrollThreshold int `long:"scroll-threshold" value-name:"N" default:"3" description:"Number of lines from the edge of screen to triger scrolling"`
		// XXX not sure how to override the defaults without overriding user options...
		//ScrollThresholdTop int `long:"scroll-threshold-top" value-name:"N" default:"3" description:"Number of lines from the top edge of screen to triger scrolling"`
		//ScrollThresholdBottom int `long:"scroll-threshold-bottom" value-name:"N" default:"3" description:"Number of lines from the bottom edge of screen to triger scrolling"`
		// XXX add named themes...
		Theme map[string]string `long:"theme" value-name:"NAME:FGCOLOR:BGCOLOR" description:"Set theme color"`
	} `group:"Configuration"`

	Introspection struct {
		ListActions bool `long:"list-actions" description:"List available actions"`
		ListThemeable bool `long:"list-themeable" description:"List available themable element names"`
		ListColors bool `long:"list-colors" description:"List usable color names"`
	} `group:"Introspection"`
}


func startup() Result {
	_, err := flags.Parse(&options)
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
	if options.Introspection.ListColors {
		for name, _ := range tcell.ColorNames {
			fmt.Println("    "+ name) }
		return OK }

	// globals...
	INPUT_FILE = options.Pos.FILE
	LIST_CMD = options.ListCommand
	TRANSFORM_CMD = options.TransformCommand

	TITLE_LINE_FMT = options.Chrome.Title
	TITLE_LINE = TITLE_LINE_FMT != ""
	TITLE_CMD = options.Chrome.TitleCommand

	STATUS_LINE_FMT = options.Chrome.Status
	STATUS_LINE = STATUS_LINE_FMT != ""
	STATUS_CMD = options.Chrome.StatusCommand

	SIZE = strings.Split(options.Chrome.Size, ",")
	ALIGN = strings.Split(options.Chrome.Align, ",")
	SPAN_MODE = options.Chrome.Span
	SPAN_LEFT_MIN_WIDTH = options.Chrome.SpanLeftMin
	SPAN_RIGHT_MIN_WIDTH = options.Chrome.SpanRightMin
	SPAN_SEPARATOR = []rune(options.Chrome.SpanSeparator)[0]
	OVERFLOW_INDICATOR = []rune(options.Chrome.OverflowIndicator)[0]
	TAB_SIZE = options.Chrome.Tab

	SCROLL_THRESHOLD_TOP = options.Config.ScrollThreshold
	SCROLL_THRESHOLD_BOTTOM = options.Config.ScrollThreshold

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
