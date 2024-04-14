/*
* TODO (stage 1: basics):
*	- basic navigation -- DONE
*	- keybindings -- DONE
*	- shell commands
*		- shared state (env) -- DONE
*		- startup
*			pipe -- DONE
*			-c CMD
*		- update (action: re-run startup command)
*		- action
*			keybinding -- DONE
*			-key:<key> CMD
*		- transform (line -> screen line)
*		- output (???)
*
*
* TODO (stage 2: features):
*	- CLI flags/API
*	- UI:
*		- status
*		- title
*		- borders
*	- config / defaults
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
import "path"
import "fmt"
//import "flag"
import "log"
import "bytes"
import "strings"
import "unicode"
import "bufio"
import "reflect"
import "regexp"

import "github.com/gdamore/tcell/v2"


// XXX need to account 
var SHELL = "bash -c"

var TAB_SIZE = 8

var ROWS, COLS int

var COL_OFFSET = 0
var ROW_OFFSET = 0

// current row relative to viewport...
var CURRENT_ROW = 0
var CURRENT_ROW_BUF []rune

var SCROLL_THRESHOLD_TOP = 3
var SCROLL_THRESHOLD_BOTTOM = 3

// XXX cursor mode...
//		- cursor
//		- line
//		- page
//		- pattern


var TEXT_BUFFER_WIDTH = 0
var TEXT_BUFFER = [][]rune{ {} }

var SELECTION_BUFFER = [][]rune{}

// XXX load this from config...
// XXX how do we represent other keys???
var KEYBINDINGS = map[string]string {
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

	"x": "X=! ls -l \"$LINE\"",
	"a": "A=! A=${A:-1} echo $(( A + 1 ))",
	"w": "! echo $A >> sum.log",

	// XXX test non-existing method...
	"Insert": "Moo",
}


func file2buffer(file *os.File){

	// XXX set this to a logical value...
	CURRENT_ROW = 0
	TEXT_BUFFER = [][]rune{}
	n := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan(){
		line := scanner.Text()
		TEXT_BUFFER = append(TEXT_BUFFER, []rune{})
		var i int
		for i = 0 ; i < len(line) ; i++ {
			TEXT_BUFFER[n] = append(TEXT_BUFFER[n], rune(line[i])) }
		// set max line width...
		if i > TEXT_BUFFER_WIDTH {
			TEXT_BUFFER_WIDTH = i }
		n++ }
	// keep at least one empty line in buffer...
	// XXX should we do this here or in the looping code???
	if n == 0 {
		TEXT_BUFFER = append(TEXT_BUFFER, []rune{}) } }


// XXX not sure if we need style arg here...
func drawScreen(screen tcell.Screen, style tcell.Style){
	// XXX
	screen.Clear()
	var col, row int
	for row = 0 ; row < ROWS ; row++ {
		var buf_row = row + ROW_OFFSET
		var col_offset = 0

		if(CURRENT_ROW == row){
			CURRENT_ROW_BUF = TEXT_BUFFER[buf_row] }

		for col = 0 ; col < COLS ; col++ {
			var buf_col = col + COL_OFFSET

			// mark current row...
			// XXX line mode...
			// XXX need to hide cursor...
			if CURRENT_ROW == row {
				style = style.Reverse(true)
			} else {
				style = style.Reverse(false) }

			// normalize...
			line := []rune{}
			if buf_row < len(TEXT_BUFFER) {
				line = TEXT_BUFFER[buf_row] }
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
	if CURRENT_ROW < ROWS-1 && CURRENT_ROW + ROW_OFFSET < len(TEXT_BUFFER)-1 && 
			// account for SCROLL_THRESHOLD_BOTTOM
			(CURRENT_ROW < ROWS - SCROLL_THRESHOLD_BOTTOM - 1 ||
				ROW_OFFSET + ROWS == len(TEXT_BUFFER)) {
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
	offset := len(TEXT_BUFFER) - ROWS 
	if ROW_OFFSET == offset {
		CURRENT_ROW = ROWS - 1
	} else {
		ROW_OFFSET = len(TEXT_BUFFER) - ROWS }
	return true }

// horizontal navigation...
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

// actions...
func (this Actions) Update() bool {
	// XXX re-run startup command in curent env...
	return true }

var ACTIONS Actions

var ENV = map[string]string {}

//func buildEnv(){}
//func readEnv(){}

var isVarCommand = regexp.MustCompile(`^[a-zA-Z_]+=`)

func callAction(action string) bool {
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
		return true }

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
		env := []string{}
		for k, v := range ENV {
			env = append(env, k +"="+ v) }
		cmd.Env = append(cmd.Environ(), 
			append(env,
				"LINE="+ string(CURRENT_ROW_BUF[:]))...)

		// run the command...
		// XXX this should be async???
		//		...option??
		if err := cmd.Run(); err != nil {
			log.Println("Error executing: \""+ code +"\":", err) 
			// XXX should we break here???
			return true }

		// handle output...
		if prefix == '@' {
			// XXX ...

		} else if prefix == '!' {
			// XXX read stdout int env...

		} else if prefix == '<' {
			// XXX pass stdout to file2buffer(..)...
		}

		// handle env...
		if name != "" {
			ENV[name] = stdout.String() } 

	// ACTION...
	} else {
		method := reflect.ValueOf(&ACTIONS).MethodByName(action)
		// test if action exists....
		if ! method.IsValid() {
			log.Println("Error: Unknown action:", action) 
			return true }
		res := method.Call([]reflect.Value{}) 
		// exit if action returns false...
		if value, ok := res[0].Interface().(bool) ; ok && !value  {
			return false } }
	return true }
func callHandler(key string) bool {
	if action, exists := KEYBINDINGS[key] ; exists {
		return callAction(action) }
	return true }


func evt2keySeq(evt tcell.EventKey) []string {
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


func fm(){
	defStyle := tcell.StyleDefault.
		Background(tcell.ColorReset).
		Foreground(tcell.ColorReset)
	//boxStyle := tcell.StyleDefault.
	//	Foreground(tcell.ColorWhite).
	//	Background(tcell.ColorPurple)

	// setup...
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err) }
	if err := screen.Init(); err != nil {
		log.Fatalf("%+v", err) }
	screen.SetStyle(defStyle)
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

	// XXX handle args...
	// XXX

	// file...
	if len(os.Args) > 1 {
		file, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Println(err)
			return }
		defer file.Close()
		file2buffer(file) 
	// pipe...
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			log.Fatalf("%+v", err) }
		if stat.Mode() & os.ModeNamedPipe != 0 {
			// XXX do we need to close this??
			//defer os.Stdin.Close()
			file2buffer(os.Stdin) } }

	for {

		COLS, ROWS = screen.Size()

		// XXX rename...
		drawScreen(screen, defStyle)

		screen.Show()

		evt := screen.PollEvent()

		switch evt := evt.(type) {
			case *tcell.EventResize:
				// keep the selection in the same spot...
				COLS, ROWS = screen.Size()
				offset := ROWS - SCROLL_THRESHOLD_BOTTOM - 1 
				// bottom...
				if CURRENT_ROW > offset {
					// if window too small keep selection in the middle...
					if ROWS < SCROLL_THRESHOLD_TOP + SCROLL_THRESHOLD_BOTTOM {
						// make this proportional...
						r := (SCROLL_THRESHOLD_TOP + SCROLL_THRESHOLD_BOTTOM + 1) / SCROLL_THRESHOLD_BOTTOM
						offset = ROWS / r }
					delta := CURRENT_ROW - offset
					// move selection and content together...
					CURRENT_ROW = offset
					ROW_OFFSET += delta 
				} else {
					// XXX do we need this???
					screen.Sync() }

			case *tcell.EventKey:
				for _, key := range evt2keySeq(*evt) {
					if ! callHandler(key) {
						return } }
				// defaults...
				if evt.Key() == tcell.KeyEscape || evt.Key() == tcell.KeyCtrlC {
					return }

			case *tcell.EventMouse:
				buttons := evt.Buttons()
				// XXX handle double click...
				// XXX handle modifiers...
				if buttons & tcell.Button1 != 0 || buttons & tcell.Button2 != 0 {
					_, CURRENT_ROW = evt.Position()

				} else if buttons & tcell.WheelUp != 0 {
					if ! callHandler("WheelUp") {
						return }

				} else if buttons & tcell.WheelDown != 0 {
					if ! callHandler("WheelDown") {
						return } }

		} } }


func main(){
    // open log file
    logFile, err := os.OpenFile(
		"./"+ path.Base(os.Args[0]) +".log", 
		os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        log.Panic(err) }
    defer logFile.Close()
    // Set log out put and enjoy :)
    log.SetOutput(logFile)

	fm() }

// vim:set sw=4 ts=4 nowrap :
