
package main

import "os"
import "fmt"

import "github.com/nsf/termbox-go"
import "github.com/mattn/go-runewidth"

var TAB_SIZE = 8

var ROWS, COLS int
var offsetX, offsetY int



var text_buffer = [][]rune {
	{'h','e','l','l','o',},
	{'\t','w','o','r','l','d','\t','!',},
}
func display_text_buffer(){
	var col, row int
	var col_offset = 0
	for row = 0 ; row < ROWS ; row++ {
		var buf_row = row + offsetY
		for col = 0 ; col < COLS ; col++ {
			var buf_col = col + offsetX
			// XXX can't break lines before an operator???!!
			if buf_row >= 0 && buf_row < len(text_buffer) && 
					buf_col >= 0 && buf_col < len(text_buffer[buf_row]) {

				// XXX need to handle escape sequences (basic state machine -- set bg/fg/...) (???)

				// tab -- offset output to next tabstop... 
				if text_buffer[buf_row][buf_col] == '\t' {
					col_offset += TAB_SIZE - (col % TAB_SIZE)
					if col_offset == 0 {
						col_offset = TAB_SIZE
					}

				// normal characters...
				} else {
					termbox.SetChar(col + col_offset, row, text_buffer[buf_row][buf_col])
				}
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
		} }

}

func main(){
	run_fm() }

// vim:set sw=4 ts=4 :
