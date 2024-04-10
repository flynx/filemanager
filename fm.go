/*
* TODO:
*	- buffer:
*		- from stdin
*		- from command
*		- non-blocking update
*		- keep position on update
*			- wait for cur-line in update buffer
*			- redraw relative to current line
*	- key bindings:
*		- reasonable defaults
*		- config
*		- action 
*	- navigation:
*		- cursor / line / pattern
*	- 
*
*/

package main

import "os"
import "fmt"

import "github.com/nsf/termbox-go"
import "github.com/mattn/go-runewidth"



var TAB_SIZE = 8

var ROWS, COLS int

var COL_OFFSET = 0
var ROW_OFFSET = 0

var CURRENT_ROW= 0
var CURRENT_ROW_BUF []rune

// XXX cursor mode...
//		- cursor
//		- line
//		- pattern


var TEXT_BUFFER = [][]rune {
	{'h','e','l','l','o',},
	{'\t','w','o','r','l','d','\t','!',},
	{},
	{},
	{'e','n','d','.'},
}
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
						col_offset = TAB_SIZE
					}

				// normal characters...
				} else {
					termbox.SetChar(col + col_offset, row, TEXT_BUFFER[buf_row][buf_col]) }
			} 
		}
	}
}



func print_msg(col, row int, msg string){
	for _, c := range msg {
		termbox.SetChar(col, row, c)
		col += runewidth.RuneWidth(c)
	}
}

func run_fm(){
	err := termbox.Init()
	if err != nil {
		fmt.Println(err)
		os.Exit(1) }


	for {
		COLS, ROWS = termbox.Size()

		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		display_text_buffer()

		termbox.Flush()

		evt := termbox.PollEvent()
		if evt.Type == termbox.EventKey {
			// Esc -> exit
			if  evt.Key == termbox.KeyEsc {
				termbox.Close() 
				break } 

			if evt.Key == termbox.KeyArrowUp {
				// XXX option to skip rows at top...
				if CURRENT_ROW > 0 {
					CURRENT_ROW-- } 
				// XXX scroll the buffer...
			}

			if evt.Key == termbox.KeyArrowDown {
				if CURRENT_ROW < ROWS-1 && CURRENT_ROW + ROW_OFFSET < len(TEXT_BUFFER)-1 {
					CURRENT_ROW++ } 
				// XXX scroll the buffer...
			}
		} }

}

func main(){
	run_fm() }

// vim:set sw=4 ts=4 :
