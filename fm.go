
package main

import "os"
import "fmt"

import "github.com/nsf/termbox-go"
import "github.com/mattn/go-runewidth"


func print_msg(col, line int, msg string){
	for _, c := range msg {
		termbox.SetChar(col, line, c)
		col += runewidth.RuneWidth(c)
	}
}

func run_fm(){
	err := termbox.Init()
	if err != nil {
		fmt.Println(err)
		os.Exit(1) }

	print_msg(10, 10, "Hello World")
	termbox.Flush()
	termbox.PollEvent()


	termbox.Close()
}

func main(){
	run_fm()
}

// vim:set sw=4 ts=4 :
