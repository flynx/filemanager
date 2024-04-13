/*
* TODO:
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
import "fmt"
import "log"
import "strings"
import "unicode"
import "bufio"
import "reflect"

import "github.com/gdamore/tcell/v2"



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

	// XXX test non-existing method...
	"Insert": "Moo",
}


func file2buffer(filename string){
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return }

	defer file.Close()
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

		//if(CURRENT_ROW == row){
		//	CURRENT_ROW_BUF = TEXT_BUFFER[buf_row] }

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

var ACTIONS Actions

func callAction(name string) bool {
	// builtin actions...
	if name == "Exit" {
		return false }
	// actions...
	method := reflect.ValueOf(&ACTIONS).MethodByName(name)
	// test if action exists....
	if ! method.IsValid() {
		log.Println("Error: Unknown action:", name) 
		return true }
	res := method.Call([]reflect.Value{}) 
	// exit if action returns false...
	if value, ok := res[0].Interface().(bool) ; ok && !value  {
		return false } 
	// only the first match is valid -- ignore the rest...
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
	} else if k > tcell.KeyRune {
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

	// args...
	// XXX
	if len(os.Args) > 1 {
		file2buffer(os.Args[1]) }

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
	fm() }

// vim:set sw=4 ts=4 nowrap :
