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
//import "strconv"
//import "unicode"
import "bufio"

import "reflect"

import "github.com/nsf/termbox-go"
import "github.com/mattn/go-runewidth"



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


// XXX specify a box/cell to draw in...
func display_text_buffer(){
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
				// XXX make style configurable...
				termbox.SetBg(col, row, termbox.ColorDefault | termbox.AttrReverse | termbox.AttrBold)
			} else {
				termbox.SetBg(col, row, termbox.ColorDefault) }

			// XXX can't break lines before an operator???!!
			if buf_row >= 0 && buf_row < len(TEXT_BUFFER) && 
					buf_col >= 0 && buf_col < len(TEXT_BUFFER[buf_row]) {

				// XXX handle escape sequences (basic state machine -- set bg/fg/...)???

				// tab -- offset output to next tabstop... 
				if TEXT_BUFFER[buf_row][buf_col] == '\t' {
					col_offset += TAB_SIZE - (col % TAB_SIZE)
					if col_offset == 0 {
						col_offset = TAB_SIZE }

				// normal characters...
				} else {
					termbox.SetChar(col + col_offset, row, TEXT_BUFFER[buf_row][buf_col]) } } } } }



func print_msg(col, row int, msg string){
	for _, c := range msg {
		termbox.SetChar(col, row, c)
		col += runewidth.RuneWidth(c) } }


type Cell struct {
	top int
	left int
	bottom int
	right int
	cols int
	rows int

	// XXX spec fg, bg, border...
}


// XXX since termbox is global, is there a point in holding any local 
//		data here???
// XXX can this be a map???
type Actions struct {}

// vertical navigation...
func (this Actions) Up() bool {
	// XXX option to skip rows at top...
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

// XXX load this from config...
// XXX how do we represent other keys???
var KEYBINDINGS = map[termbox.Key]string {
	termbox.KeyEsc: "Exit",
	termbox.KeyArrowUp: "Up",
	termbox.KeyArrowDown: "Down",
	// XXX STUB -- change keys...
	termbox.KeyArrowLeft: "ScrollUp",
	termbox.KeyArrowRight: "ScrollDown",

	termbox.KeyPgup: "PageUp",
	termbox.KeyPgdn: "PageDown",
	termbox.KeyHome: "Top",
	termbox.KeyEnd: "Bottom",

	// XXX test non-existing method...
	termbox.KeyInsert: "Moo",
}


// NOTE: this mirrors termbox's Key*/Mouse* constants (0xFFFF - evt.Key = index)...
/*
var key_map = []string{
	"F1", "F2", "F3", "F4", "F5", "F6", 
	"F7", "F8", "F9", "F10", "F11", "F12",
	"Insert", "Delete", "Home", "End", 
	// XXX add aliases...
	"PgUp", "PgDown",
	"Up", "Down", "Left", "Right",
	// mouse...
	"MouseLeft", "MouseMiddle", "MouseRight",
	"MouseRelease",
	"MouseWheelUp", "MouseWheelDown",
}
func evtKey2Seq(evt termbox.Event) []string {
	key_seq := []string{}

	// XXX form key...
	//		- evt.Key		- key constants (special keys) / 0
	//		- evt.Ch		- rune / 0
	//		- evt.Mod		- modifiers / 0
	//		form a list of candidates (as in keyboard.js) and 
	//		test each in order, e.g.:
	//			alt-ctrl-a
	//			ctrl-alt-a
	//			ctrl-a
	//			alt-a
	//			a
	// XXX need langmap to allow input in other languages -- piggyback on vim?
	
	shift := false
	ctrl := false
	alt := evt.Mod == termbox.ModAlt
	meta := false
	var key string
	var mkey string
	switch {
		case evt.Ch != 0:
			shift = unicode.IsUpper(evt.Ch)
			// XXX ctrl???
			// XXX
			key = string(unicode.ToLower(evt.Ch))
			// XXX get ascii key -- keymap...
			if len(key) > 1 {
				mkey = "" }
		case evt.Key <= termbox.KeyF1 && evt.Key >= termbox.MouseWheelDown:
			key := key_map[0xFFFF - evt.Key]
		// XXX
	}
	//for _, key := range key_seq {
	//	// XXX
	//}

	return key_seq }
//*/


func run_fm(){
	if err := termbox.Init(); err != nil {
		fmt.Println(err)
		os.Exit(1) }

	// args...
	if len(os.Args) > 1 {
		file2buffer(os.Args[1]) }

	for {
		COLS, ROWS = termbox.Size()


		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		display_text_buffer()

		termbox.Flush()


		evt := termbox.PollEvent()
		// handle mouse...
		// XXX

		// handle keyboard...
		if evt.Type == termbox.EventKey {
			// handle key...
			if action, exists := KEYBINDINGS[evt.Key] ; exists {
				// builtin actions...
				if action == "Exit" {
					termbox.Close()
					break }

				// actions...
				method := reflect.ValueOf(&ACTIONS).MethodByName(action)
				// test if action exists....
				if ! method.IsValid() {
					// XXX report error...
					continue }
				res := method.Call([]reflect.Value{}) 
				// exit if action returns false...
				if value, ok := res[0].Interface().(bool) ; ok && !value  {
					break } } } }

}

func main(){
	run_fm() }

// vim:set sw=4 ts=4 :
