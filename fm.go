/*
* TODO (stage 1: basics):
*	- basic navigation -- DONE
*	- keybindings -- DONE
*	- shell commands
*		- shared state (env) -- DONE
*		- startup
*			pipe -- DONE
*			-c CMD -- DONE
*		- update (action: re-run startup command)
*		- action
*			keybinding -- DONE
*			-key:<key>:<CMD>
*		- transform (line -> screen line)
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
import "strings"
import "unicode"
import "bufio"
import "reflect"
import "regexp"

import "github.com/jessevdk/go-flags"
import "github.com/gdamore/tcell/v2"


var LIST_CMD string
var INPUT_FILE string
var OUTPUT_STR string

// XXX need to account 
var SHELL = "bash -c"

var TAB_SIZE = 8

var ROWS, COLS int
var CONTENT_ROWS, CONTENT_COLS int

var COL_OFFSET = 0
var ROW_OFFSET = 0

var SCROLLBAR = false
var SCROLLBAR_FG = tcell.RuneCkBoard
var SCROLLBAR_BG = tcell.RuneBoard

var SCROLL_THRESHOLD_TOP = 3
var SCROLL_THRESHOLD_BOTTOM = 3

// XXX
//var STATUS_LINE = false

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
	text string
}
//var TEXT_BUFFER = [][]rune{ {} }
var TEXT_BUFFER = []Row{}

var SELECTION = []string{}

// XXX load this from config...
// XXX how do we represent other keys???
type Keybindings map[string]string
var KEYBINDINGS = Keybindings {
	"Esc": "Exit",
	"q": "Exit",
	//"Q": "Exit",
	//"shift+q": "Exit",

	"Up": "Up",
	"Down": "Down",

	"WheelUp": "ScrollUp",
	"WheelDown": "ScrollDown",

	"PgUp": "PageUp",
	"PgDn": "PageDown",
	"Home": "Top",
	"End": "Bottom",

	"Enter": "! echo \"$LINE\" >> moo.log",
	// XXX should we also have a "Click" event
	//"ClickSelected": "Exit",

	"x": "! echo \"$SELECTION\" > selection",

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
// XXX not sure if we need style arg here...
// XXX add scrollbar...
func drawScreen(screen tcell.Screen, theme Theme){
	screen.Clear()

	// scrollbar...
	var scroller_size, scroller_offset int
	SCROLLBAR = len(TEXT_BUFFER) > ROWS
	if SCROLLBAR {
		r := float32(ROWS) / float32(len(TEXT_BUFFER))
		scroller_size = 1 + int(float32(ROWS - 1) * r)
		scroller_offset = int(float32(ROW_OFFSET + 1) * r) }

	var col, row int
	style := theme["default"]
	for row = 0 ; row < ROWS ; row++ {
		var buf_row = row + ROW_OFFSET

		// row theming...
		style = theme["default"]
		if buf_row < len(TEXT_BUFFER) {
			// current+selected...
			if TEXT_BUFFER[buf_row].selected &&
					CURRENT_ROW == row {
				style = theme["current-selected"]
			// mark selected...
			} else if TEXT_BUFFER[buf_row].selected {
				style = theme["selected"]
			// current...
			} else if CURRENT_ROW == row {
				style = theme["current"] } } 

		// normalize...
		line := []rune{}
		if buf_row < len(TEXT_BUFFER) {
			line = []rune(TEXT_BUFFER[buf_row].text) }

		var col_offset = 0
		for col = 0 ; col < COLS ; col++ {
			// scrollbar...
			if SCROLLBAR && col == COLS-1 {
				c := SCROLLBAR_BG
				if row >= scroller_offset && row < scroller_offset+scroller_size {
					c = SCROLLBAR_FG }
				screen.SetContent(col + col_offset, row, c, nil, theme["default"])
				continue }

			var buf_col = col + COL_OFFSET

			// get rune...
			c := ' '
			if buf_col < len(line) {
				c = line[buf_col] } 

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

// XXX horizontal navigation...
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

var ACTIONS Actions

var ENV = map[string]string {}
var isVarCommand = regexp.MustCompile(`^[a-zA-Z_]+=`)
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

		// !ACTION | <ACTION | @ACTION...
		if action[0] == '!' || action[0] == '<' || action[0] == '@' {
			prefix, code := action[0], action[1:]
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			shell := strings.Fields(SHELL)
			// XXX this is ugly, split slice and unpack instead of just unpack...
			cmd := exec.Command(shell[0], append(shell[1:], code)...)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			// pass data to command via env...
			// XXX handle this globally/func...
			// SELECTED...
			selected := ""
			if TEXT_BUFFER[CURRENT_ROW + ROW_OFFSET].selected {
				selected = "1" }
			state := map[string]string {
				"SELECTED": selected,
				"SELECTION": strings.Join(SELECTION, "\n"),
				"COLS": string(CONTENT_COLS),
				"ROWS": string(CONTENT_ROWS),
				"LINES": string(len(TEXT_BUFFER)),
				"LINE": string(ROW_OFFSET + CURRENT_ROW),
				"TEXT": TEXT_BUFFER[CURRENT_ROW].text,
			}
			env := []string{}
			for k, v := range ENV {
				// XXX go: really ugly...
				if v == string(0) {
					v = "0" }
				if v != "" {
					env = append(env, k +"="+ v) } }
			for k, v := range state {
				// XXX go: really ugly...
				if v == string(0) {
					v = "0" }
				if v != "" {
					env = append(env, k +"="+ v) } }
			cmd.Env = append(cmd.Environ(), env...) 

			// run the command...
			// XXX this should be async???
			//		...option??
			if err := cmd.Run(); err != nil {
				log.Println("Error executing: \""+ code +"\":", err) 
				//log.Println("    ENV:", env) 
				break }

			// handle output...
			if prefix == '@' {
				OUTPUT_STR += stdout.String()

			} else if prefix == '!' {
				// XXX read stdout into env... (???)
				//env := strings.Split(stdout.String(), "\n")

			} else if prefix == '<' {
				// XXX pass stdout to file2buffer(..)...
				// XXX stdout should be read line by line as it comes...
				// XXX keep selection and current item and screen position 
				//		relative to current..
				// XXX STUB??
				str2buffer(stdout.String()) }

			// handle env...
			if name != "" {
				ENV[name] = stdout.String() } 

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
// XXX add alias support...
func callHandler(key string) bool {
	if action, exists := KEYBINDINGS[key] ; exists {
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

func fm(){
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
		COLS, ROWS = screen.Size()


		CONTENT_COLS, CONTENT_ROWS = COLS, ROWS
		// XXX also handle borders titlebar and statusbar...
		if SCROLLBAR {
			CONTENT_COLS -= 1 }

		// XXX rename...
		drawScreen(screen, THEME)

		screen.Show()

		evt := screen.PollEvent()

		switch evt := evt.(type) {
			// XXX BUG: resizing smaller with last row in TEXT_BUFFER selected
			//		the offset jumps by +/- 3...
			// keep the selection in the same spot...
			case *tcell.EventResize:
				COLS, ROWS = screen.Size()
				handleScrollLimits()

			case *tcell.EventKey:
				for _, key := range evt2keys(*evt) {
					if ! callHandler(key) {
						return } }
				//log.Println("KEY:", evt.Name())
				// defaults...
				if evt.Key() == tcell.KeyEscape || evt.Key() == tcell.KeyCtrlC {
					return }

			// XXX clicking above top threshold or below bottom threshold 
			//		should scroll the cursor to the threshold...
			case *tcell.EventMouse:
				buttons := evt.Buttons()
				// XXX handle double click...
				// XXX handle modifiers...
				if buttons & tcell.Button1 != 0 || buttons & tcell.Button2 != 0 {
					col, row := evt.Position()
					// scrollbar...
					// XXX sould be nice if we started in the scrollbar 
					//		to heep handling the drag untill released...
					//		...for this to work need to either detect 
					//		drag or release...
					if SCROLLBAR && col == COLS-1 {
						ROW_OFFSET = 
							int((float32(row) / float32(ROWS - 1)) * 
							float32(len(TEXT_BUFFER) - ROWS))
					// second click in curent row...
					// XXX should we have a timeout here???
					// XXX this triggers on drag... is this a bug???
					} else if row == CURRENT_ROW {
						if ! callHandler("ClickSelected") {
							return }
					// below list...
					} else if row > len(TEXT_BUFFER) {
						CURRENT_ROW = len(TEXT_BUFFER) - 1
					// list...
					} else {
						CURRENT_ROW = row }
					handleScrollLimits()


				} else if buttons & tcell.WheelUp != 0 {
					if ! callHandler("WheelUp") {
						return }

				} else if buttons & tcell.WheelDown != 0 {
					if ! callHandler("WheelDown") {
						return } }

		} } }



// command line args...
var options struct {
	// XXX

	// XXX not used...
	ListCommand string `short:"c" long:"cmd" value-name:"CMD" env:"CMD" description:"List command"`

	// XXX chicken-egg: need to first parse the args then parse the ini 
	//		and then merge the two...
	//ArgsFile string `long:"args-file" value-name:"FILE" env:"ARGS" description:"Arguments file"`

	LogFile string `short:"l" long:"log" value-name:"FILE" env:"LOG" description:"Log file"`
	Pos struct {
		FILE string
	} `positional-args:"yes"`
}


func main(){
	_, err := flags.Parse(&options)
	if err != nil {
		if flags.WroteHelp(err) {
			return }
		log.Println("Error:", err)
		os.Exit(1) }

	// globals...
	INPUT_FILE = options.Pos.FILE
	LIST_CMD = options.ListCommand

	// log...
	if options.LogFile != "" {
		logFile, err := os.OpenFile(
			options.LogFile,
			os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Panic(err) }
		defer logFile.Close()
		// Set log out put and enjoy :)
		log.SetOutput(logFile) }

	// startup...
	fm() 

	// output...
	// XXX should this be here or in fm(..)
	if OUTPUT_STR != "" {
		fmt.Println(OUTPUT_STR) } }



// vim:set sw=4 ts=4 nowrap :
