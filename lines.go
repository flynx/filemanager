/*
* TODO (stage 1: basics):
*	- basic navigation -- DONE
*	- keybindings -- DONE
*	- shell commands
*		- shared state (env) -- DONE
*		- startup
*			pipe -- DONE
*			-c CMD -- DONE
*		- update (action: re-run startup command) -- DONE
*		- action
*			keybinding -- DONE
*			-key:<key>:<CMD>
*		- transform (line -> screen line)
*			XXX this is not needed as it is simpler ot make this part of the -c...
*				i.e.
*					lines --cmd ls --transform 'sed "s/moo/foo/"'
*				vs.
*					lines --cmd 'ls | sed "s/moo/foo/"'
*		- output (???)
*	- selection -- DONE
*	- config file / defaults
*
*
* TODO (stage 2: features):
*	- CLI flags/API
*	- UI:
*		- status
*		- title
*		- borders
*		- colors (???)
*
* XXX should we have search???
*		...can we pigiback off grep?? =)
*
*
*
* Data flow
*	list
*		filter
*		transform (line)
*
*	- buffer:
*		- from stdin
*		- from command
*		- non-blocking update
*		- keep position on update
*			- wait for cur-line in update buffer
*			- redraw relative to current line
*		- command to filter/format line on update (cursor in/out/...)
*	- key bindings:
*		- reasonable defaults
*		- config
*		- action 
*	- navigation:
*		- cursor -- a-la vim
*		- line -- a-la FAR
*		- page -- a-la more/less
*		- pattern -- a-la info
*	- selection
*	- copy/paste
*	- cells
*
* XXX need a way to show a box over the curent terminal content...
* XXX might be fun to add an inline mode -- if # of lines is less that 
*		term height Println(..) them and then play with that region of 
*		the terminal, otherwise open normally...
*
*/

package main

import "os"
import "os/exec"
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

import "github.com/jessevdk/go-flags"
import "github.com/gdamore/tcell/v2"


// XXX refactoring -- not sure about this yet...
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

var ROWS, COLS int
//var CONTENT_ROWS, CONTENT_COLS int

var COL_OFFSET = 0
var ROW_OFFSET = 0

var SCROLLBAR = false
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
	"ctrl+i": "SelectInverse",
	"ctrl+d": "SelectNone",
}


// XXX make this config-ready -- i.e. map[string]string
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

func populateLine(str string, cmd string) string {
	// %CMD...
	if strings.Contains(str, "%CMD") {
		s, err := "", error(nil)
		if cmd != "" {
			s, err = callTransform(cmd, str)
			if err != nil {
				s = "" } }
		str = strings.ReplaceAll(str, "%CMD", s) } 
	// %INDEX...
	str = strings.ReplaceAll(str, "%INDEX", 
		fmt.Sprint(ROW_OFFSET + CURRENT_ROW + 1))
	// %LINES...
	str = strings.ReplaceAll(str, "%LINES", 
		fmt.Sprint(len(TEXT_BUFFER)))
	// %SELECTED...
	current := TEXT_BUFFER[CURRENT_ROW + ROW_OFFSET]
	selected := ""
	if current.selected {
		selected = "*" }
	str = strings.ReplaceAll(str, "%SELECTED", selected)
	// %SELECTION...
	selection := fmt.Sprint(len(SELECTION))
	if selection == "0" {
		selection = "" }
	str = strings.ReplaceAll(str, "%SELECTION", selection)
	// %REST...
	rest := ""
	if len(current.text) > COLS {
		rest = current.text[COLS:] }
	str = strings.ReplaceAll(str, "%REST", rest)
	// %SPAN / fit width...
	// contract...
	if len(str) > COLS {
		overflow := (len(str) - COLS) + 3
		parts := strings.SplitN(str, "%SPAN", 2)
		str = string(parts[0][:len(parts[0])-overflow]) + "..."
		if len(parts) > 1 {
			str += parts[1] }
	// expand...
	} else {
		// XXX need a way to indicate the character to use for expansion...
		str = strings.ReplaceAll(str, "%SPAN", 
			fmt.Sprintf("%"+fmt.Sprint(COLS - len(str) + 5) +"v", "")) }
	return str }

func drawScreen(screen tcell.Screen, theme Theme){
	screen.Clear()

	// scrollbar...
	var scroller_size, scroller_offset int
	scroller_style, ok := theme["scrollbar"]
	if ! ok {
		scroller_style = theme["default"] }
	SCROLLBAR = len(TEXT_BUFFER) > ROWS
	if SCROLLBAR {
		r := float32(ROWS) / float32(len(TEXT_BUFFER))
		scroller_size = 1 + int(float32(ROWS - 1) * r)
		scroller_offset = int(float32(ROW_OFFSET + 1) * r) }

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
	for row = 0 ; row < height ; row++ {
		var buf_row = row - top_offset + ROW_OFFSET

		// row theming...
		style = theme["default"]
		non_default_style := true
		if buf_row >= 0 && 
				buf_row < len(TEXT_BUFFER) {
			// current+selected...
			if TEXT_BUFFER[buf_row].selected &&
					CURRENT_ROW == row - top_offset {
				style, non_default_style = theme["current-selected"]
			// mark selected...
			} else if TEXT_BUFFER[buf_row].selected {
				style, non_default_style = theme["selected"]
			// current...
			} else if CURRENT_ROW == row - top_offset {
				style, non_default_style = theme["current"] } } 

		// normalize...
		line := []rune{}
		// buffer line...
		if buf_row >= 0 && 
				buf_row < len(TEXT_BUFFER) && 
				row >= top_offset &&
				row <= ROWS {
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
					row == 0 {
				str = TITLE_LINE_FMT
				style, non_default_style = theme["title-line"]
				if TITLE_CMD != "" {
					cmd = TITLE_CMD }
			// status...
			} else if STATUS_LINE && 
					row == height-1 {
				str = STATUS_LINE_FMT
				style, non_default_style = theme["status-line"] 
				if STATUS_CMD != "" {
					cmd = STATUS_CMD } }
			// populate the line...
			line = []rune(populateLine(str, cmd)) }

		// set default style...
		if ! non_default_style {
			style = theme["default"] }

		var col_offset = 0
		var buf_offset = 0
		for col = 0 ; col < COLS ; col++ {
			// scrollbar...
			if SCROLLBAR && 
					col == COLS-1 &&
					! (TITLE_LINE &&
						row < top_offset) &&
					! (STATUS_LINE &&
						row == height-1) {
				c := SCROLLBAR_BG
				if row-top_offset >= scroller_offset && 
						row-top_offset < scroller_offset+scroller_size {
					c = SCROLLBAR_FG }
				screen.SetContent(col + col_offset, row, c, nil, scroller_style)
				continue }

			var buf_col = col + buf_offset + COL_OFFSET 

			// get rune...
			c := ' '
			if buf_col < len(line) {
				c = line[buf_col] } 

			// handle escape sequences...
			// see: 
			//	https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797 
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

			// tab -- offset output to next tabstop... 
			if c == '\t' {
				col_offset += TAB_SIZE - (col % TAB_SIZE)
				for i := 0 ; i <= col_offset ; i++ {
					screen.SetContent(col+i, row, ' ', nil, style) }

			// normal characters...
			} else {
				screen.SetContent(col + col_offset, row, c, nil, style) } } } }


// Actions...
// XXX since termbox is global, is there a point in holding any local 
//		data here???
// XXX can this be a map???
type Actions struct {}

// vertical navigation...
// XXX changing only CURRENT_ROW can be donwe by redrawing only two lines...
func (this Actions) Up() bool {
	if CURRENT_ROW > 0 && 
			// account for SCROLL_THRESHOLD_TOP...
			(CURRENT_ROW > SCROLL_THRESHOLD_TOP ||
				ROW_OFFSET == 0) {
		CURRENT_ROW-- 
	// scroll the buffer...
	} else {
		this.ScrollUp() }
	return true }
func (this Actions) Down() bool {
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
	return true }

// XXX should these track CURRENT_ROW relative to screen (current) or 
//		relative to content???
func (this Actions) ScrollUp() bool {
	if ROW_OFFSET > 0 {
		ROW_OFFSET-- }
	return true }
func (this Actions) ScrollDown() bool {
	if ROW_OFFSET + ROWS < len(TEXT_BUFFER) {
		ROW_OFFSET++ } 
	return true }

func (this Actions) PageUp() bool {
	if ROW_OFFSET > 0 {
		ROW_OFFSET -= ROWS 
		if ROW_OFFSET < 0 {
			this.Top() } 
	} else if ROW_OFFSET == 0 {
		this.Top() } 
	return true }
func (this Actions) PageDown() bool {
	if len(TEXT_BUFFER) < ROWS {
		CURRENT_ROW = len(TEXT_BUFFER) - 1
		return true }
	offset := len(TEXT_BUFFER) - ROWS
	if ROW_OFFSET < offset {
		ROW_OFFSET += ROWS 
		if ROW_OFFSET > offset {
			this.Bottom() } 
	} else if ROW_OFFSET == offset {
		this.Bottom() } 
	return true }

func (this Actions) Top() bool {
	if ROW_OFFSET == 0 {
		CURRENT_ROW = 0 
	} else {
		ROW_OFFSET = 0 }
	return true }
func (this Actions) Bottom() bool {
	if len(TEXT_BUFFER) < ROWS {
		CURRENT_ROW = len(TEXT_BUFFER) - 1
		return true }
	offset := len(TEXT_BUFFER) - ROWS 
	if ROW_OFFSET == offset {
		CURRENT_ROW = ROWS - 1
	} else {
		ROW_OFFSET = len(TEXT_BUFFER) - ROWS }
	return true }

/*// XXX horizontal navigation...
func (this Actions) Left() bool {
	// XXX
	return true }
func (this Actions) Right() bool {
	// XXX
	return true }

func (this Actions) ScrollLeft() bool {
	// XXX
	return true }
func (this Actions) ScrollRight() bool {
	// XXX
	return true }

func (this Actions) LeftEdge() bool {
	// XXX
	return true }
func (this Actions) RightEdge() bool {
	// XXX
	return true }
//*/

// XXX
func (this Actions) ToLine(line int) bool {
	// XXX
	return true }

// selection...
func _updateSelection(){
	SELECTION = []string{}
	for _, row := range TEXT_BUFFER {
		if row.selected {
			SELECTION = append(SELECTION, row.text) } } }
func (this Actions) SelectToggle(rows ...int) bool {
	if len(rows) == 0 {
		rows = []int{CURRENT_ROW + ROW_OFFSET} }
	for _, i := range rows {
		if TEXT_BUFFER[i].selected {
			TEXT_BUFFER[i].selected = false 
		} else {
			TEXT_BUFFER[i].selected = true } }
	_updateSelection()
	return true }
func (this Actions) SelectAll() bool {
	for i := 0; i < len(TEXT_BUFFER); i++ {
		TEXT_BUFFER[i].selected = true } 
	_updateSelection()
	return true }
func (this Actions) SelectNone() bool {
	for i := 0; i < len(TEXT_BUFFER); i++ {
		TEXT_BUFFER[i].selected = false } 
	SELECTION = []string{}
	return true }
func (this Actions) SelectInverse() bool {
	rows := []int{}
	for i := 0 ; i < len(TEXT_BUFFER) ; i++ {
		rows = append(rows, i) }
	return this.SelectToggle(rows...) }

// XXX re-run startup command in curent env...
func (this Actions) Update() bool {
	// file...
	if INPUT_FILE != "" {
		file, err := os.Open(INPUT_FILE)
		if err != nil {
			fmt.Println(err)
			return false }
		defer file.Close()
		file2buffer(file) 
	// command...
	} else if LIST_CMD != "" {
		// XXX HACK???
		TEXT_BUFFER = []Row{ {} }
		// XXX call Update action...
		callAction("<"+ LIST_CMD)
	// pipe...
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalf("%+v", err) }
		if stat.Mode() & os.ModeNamedPipe != 0 {
			// XXX do we need to close this??
			//defer os.Stdin.Close()
			file2buffer(os.Stdin) } }
	return true }

// placeholder...
// NOTE: This is never called directly but is here for documentation...
func (this Actions) Exit() bool {
	return true }


var ACTIONS Actions

var ENV = map[string]string {}

// XXX needs revision -- fells hacky...
func callCommand(code string, stdin bytes.Buffer) (bytes.Buffer, bytes.Buffer, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	shell := strings.Fields(SHELL)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)

	cmd.Stdin = &stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// pass data to command via env...
	selected := ""
	if TEXT_BUFFER[CURRENT_ROW + ROW_OFFSET].selected {
		selected = "1" }
	state := map[string]string {
		"SELECTED": selected,
		"SELECTION": strings.Join(SELECTION, "\n"),
		// XXX need a way to tell the command the current available width...
		//"COLS": fmt.Sprint(CONTENT_COLS),
		//"ROWS": fmt.Sprint(CONTENT_ROWS),
		"LINES": fmt.Sprint(len(TEXT_BUFFER)),
		"LINE": fmt.Sprint(ROW_OFFSET + CURRENT_ROW),
		"TEXT": TEXT_BUFFER[CURRENT_ROW].text,
	}
	env := []string{}
	for k, v := range ENV {
		if v != "" {
			env = append(env, k +"="+ v) } }
	for k, v := range state {
		if v != "" {
			env = append(env, k +"="+ v) } }
	cmd.Env = append(cmd.Environ(), env...) 

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
func callAction(actions string) bool {
	// XXX make split here a bit more cleaver:
	//		- support ";"
	//		- support quoting of separators, i.e. ".. \\\n .." and ".. \; .."
	//		- ignore string literal content...
	for _, action := range strings.Split(actions, "\n") {
		action = strings.Trim(action, " \t")
		if len(action) == 0 {
			continue }
		// builtin actions...
		if action == "Exit" {
			return false }

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
		//		@ CMD	- simple command
		//		! CMD	- stdout treated as env variables, one per line
		//		< CMD	- stdout read into buffer
		//		> CMD	- stdout printed to lines stdout
		//		| CMD	- current line passed to stdin
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

			//stdout, stderr, err := callCommand(code, stdin)
			stdout, _, err := callCommand(code, stdin)
			if err != nil {
				break }

			// list output...
			if slices.Contains(prefix, '<') {
				// XXX stdout should be read line by line as it comes...
				// XXX keep selection and current item and screen position 
				//		relative to current..
				str2buffer(stdout.String()) }
			// output to stdout...
			if slices.Contains(prefix, '>') {
				STDOUT += stdout.String() }
			// output to env...
			if slices.Contains(prefix, '!') {
				for _, str := range strings.Split(stdout.String(), "\n") {
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
					STDOUT += stdout.String()
				} else {
					ENV[name] = stdout.String() } }

		// ACTION...
		} else {
			method := reflect.ValueOf(&ACTIONS).MethodByName(action)
			// test if action exists....
			if ! method.IsValid() {
				log.Println("Error: Unknown action:", action) 
				continue }
			res := method.Call([]reflect.Value{}) 
			// exit if action returns false...
			if value, ok := res[0].Interface().(bool) ; ok && !value  {
				return false } } }
	return true }
func callHandler(key string) bool {
	// expand aliases...
	seen := []string{ key }
	if action, exists := KEYBINDINGS[key] ; exists {
		_action := action
		for exists && ! slices.Contains(seen, _action) {
			if _action, exists = KEYBINDINGS[_action] ; exists {
				action = _action } }
		return callAction(action) }
	return true }


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

	// basic translation...
	if key == " " {
		key = "Space" }

	if mod & tcell.ModCtrl != 0 {
		mods = append(mods, "ctrl") }
	if mod & tcell.ModAlt != 0 {
		mods = append(mods, "alt") }
	if mod & tcell.ModMeta != 0 {
		mods = append(mods, "meta") }
	if !shifted && mod & tcell.ModShift != 0 {
		mods = append(mods, "shift") }

	// XXX generate all mod combinations...
	// XXX sort by length...
	// XXX

	// XXX STUB...
	for _, m := range mods {
		key_seq = append(key_seq, m +"+"+ key) }
	// uppercase letter...
	if shifted {
		key_seq = append(key_seq, Key) }
	key_seq = append(key_seq, key)

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
	COLS, ROWS = screen.Size() 
	if TITLE_LINE {
		ROWS-- }
	if STATUS_LINE {
		ROWS-- } }

func lines(){
	// setup...
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err) }
	if err := screen.Init(); err != nil {
		log.Fatalf("%+v", err) }
	screen.SetStyle(THEME["default"])
	screen.EnableMouse()
	screen.EnablePaste()
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
		if SCROLLBAR {
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
					if ! callHandler(key) {
						return } }
				//log.Println("KEY:", evt.Name())
				// defaults...
				if evt.Key() == tcell.KeyEscape || evt.Key() == tcell.KeyCtrlC {
					return }

			case *tcell.EventMouse:
				buttons := evt.Buttons()
				// XXX handle double click...
				// XXX handle modifiers...
				if buttons & tcell.Button1 != 0 || buttons & tcell.Button2 != 0 {
					col, row := evt.Position()
					// title/status bars...
					top_offset := 0
					if TITLE_LINE {
						top_offset = 1
						if row == 0 {
							// XXX handle titlebar click???
							continue } }
					if STATUS_LINE {
						if row == ROWS + 1 {
							// XXX handle statusbar click???
							continue } }
					// scrollbar...
					// XXX sould be nice if we started in the scrollbar 
					//		to keep handling the drag untill released...
					//		...for this to work need to either detect 
					//		drag or release...
					if SCROLLBAR && col == COLS-1 {
						ROW_OFFSET = 
							int((float32(row - top_offset) / float32(ROWS - 1)) * 
							float32(len(TEXT_BUFFER) - ROWS))
					// second click in curent row...
					// XXX should we have a timeout here???
					// XXX this triggers on drag... is this a bug???
					} else if row - top_offset == CURRENT_ROW {
						if ! callHandler("ClickSelected") {
							return }
					// below list...
					} else if row > len(TEXT_BUFFER) {
						CURRENT_ROW = len(TEXT_BUFFER) - 1
					// list...
					} else {
						CURRENT_ROW = row - top_offset }
					handleScrollLimits()


				} else if buttons & tcell.WheelUp != 0 {
					if ! callHandler("WheelUp") {
						return }

				} else if buttons & tcell.WheelDown != 0 {
					if ! callHandler("WheelDown") {
						return } } } } }



// command line args...
var options struct {
	Pos struct {
		FILE string
	} `positional-args:"yes"`

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
		Select string `short:"s" long:"select" value-name:"CMD" env:"SELECT" description:"Command to execute on item select"`
		Reject string `short:"r" long:"reject" value-name:"CMD" env:"REJECT" description:"Command to execute on reject"`
	} `group:"Actions"`

	Keyboard struct {
		Key map[string]string `short:"k" long:"key" value-name:"KEY:ACTION" description:"Bind key to action"`
		// XXX move this to help...
		ListActions bool `long:"list-actions" description:"List available actions"`
	} `group:"Keyboard"`

	Chrome struct {
		Title string `long:"title" value-name:"FMT" env:"TITLE" default:" %CMD " description:"Title format"`
		Status string `long:"status" value-name:"FMT" env:"STATUS" default:" %CMD %SPAN %INDEX/%LINES " description:"Status format"`
		TitleCommand string `long:"title-cmd" value-name:"CMD" env:"TITLE_CMD" description:"Title command"`
		StatusCommand string `long:"status-cmd" value-name:"CMD" env:"STATUS_CMD" description:"Status command"`
	} `group:"Chrome"`

	Config struct {
		LogFile string `short:"l" long:"log" value-name:"FILE" env:"LOG" description:"Log file"`
		Separator string `long:"separator" value-name:"STRING" default:"\\n" env:"SEPARATOR" description:"Command separator"`
	} `group:"Configuration"`
}


func main(){
	_, err := flags.Parse(&options)
	if err != nil {
		if flags.WroteHelp(err) {
			return }
		log.Println("Error:", err)
		os.Exit(1) }

	// doc...
	if options.Keyboard.ListActions {
		t := reflect.TypeOf(&ACTIONS)
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			fmt.Println("    "+ m.Name) }
		return }

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

	// log...
	if options.Config.LogFile != "" {
		logFile, err := os.OpenFile(
			options.Config.LogFile,
			os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Panic(err) }
		defer logFile.Close()
		// Set log out put and enjoy :)
		log.SetOutput(logFile) }

	// startup...
	lines() 

	// output...
	// XXX should this be here or in lines(..)
	if STDOUT != "" {
		fmt.Println(STDOUT) } }



// vim:set sw=4 ts=4 nowrap :
