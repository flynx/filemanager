/*
* XXX split this into two parts:
*		- tcell
*			- drawing
*			- event loop
*		- generic
*			- config
*			- cli
*			- key/click handlers
*/
package main

import (
	//"fmt"
	"log"
	"os"
	//"strings"
	//"slices"
	//"reflect"

	"github.com/gdamore/tcell/v2"
	//"github.com/jessevdk/go-flags"

	//"github.com/mkideal/cli"
)




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



type TcellDrawer struct {
	tcell.Screen

	Lines *Lines
	// XXX

	__style_cache map[string]tcell.Style
}
func (this *TcellDrawer) Setup(lines Lines) *TcellDrawer {
	this.Lines = &lines
	lines.CellsDrawer = this
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Panic(err) }
	this.Screen = screen
	if err := this.Screen.Init(); err != nil {
		log.Panic(err) }
	this.EnableMouse()
	this.EnablePaste()

	// XXX

	return this }
func (this *TcellDrawer) UpdateTheme() *TcellDrawer {
	this.__style_cache = nil
	return this }
func (this *TcellDrawer) UpdateGeometry() *TcellDrawer {

	// XXX update geometry...

	return this }
func (this *TcellDrawer) Loop() Result {
	defer this.Finalize()
	// XXX event loop ???
	for {
		this.UpdateGeometry()
		this.Lines.Draw()
		this.Show()

		evt := this.PollEvent()

		switch evt := evt.(type) {
			// XXX
			case *tcell.EventKey:
				log.Println("---", evt)
				return OK
		} }
	return OK }
// handle panics and cleanup...
func (this *TcellDrawer) Finalize() {
	maybePanic := recover()
	this.Screen.Fini()
	if maybePanic != nil {
		panic(maybePanic) } }

func Style2TcellStyle(style Style) tcell.Style {
	// full style...
	if len(style) == 1 {
		switch style[0] {
			case "reverse":
				return tcell.StyleDefault.
					Reverse(true)
			default:
				return tcell.StyleDefault } }
	// componnt style...
	res := tcell.StyleDefault
	fg, bg, _ := res.Decompose()
	if style[0] != "default" && 
			style[0] != "foreground" && 
			style[0] != "fg" {
		switch style[0] {
			case "background":
				res.Foreground(bg)
			default:
				res = res.Foreground(tcell.GetColor(style[0])) } }
	if style[1] != "default" &&
			style[0] != "background" && 
			style[0] != "bg" {
		switch style[0] {
			case "foreground":
				res.Background(fg)
			default:
				res = res.Background(tcell.GetColor(style[1])) } }
	return res }
func (this *TcellDrawer) drawCells(col, row int, str string, style_name string, style Style) {
	if style_name == "EOL" {
		return }
	//log.Printf("%2v, %-2v: %3v:%#-25v (%v)\n", col, row, len([]rune(str)), str, style_name)
	// get style (cached)...
	if this.__style_cache == nil {
		this.__style_cache = map[string]tcell.Style{} }
	s, ok := this.__style_cache[style_name]
	if ! ok {
		s := Style2TcellStyle(style)
		this.__style_cache[style_name] = s }
	for i, r := range []rune(str) {
		this.SetContent(col+i, row, r, nil, s) } }

// XXX should this take Lines ot Settings???
func NewTcellLines(l ...Lines) TcellDrawer {
	var lines Lines
	if len(l) == 0 {
		lines = Lines{}
	} else {
		lines = l[0] }

	drawer := TcellDrawer{}
	drawer.Setup(lines)

	return drawer }




func main(){
	//* XXX stub...
	lines := NewTcellLines(Lines{
		SpanMode: "*,5",
		SpanSeparator: "|",
		Width: 20,
		Height: 6,
		Border: "│┌─┐│└─┘",
	})
	lines.Lines.Append(
		"Some text",
		"Current",
		"Some%SPANColumns")
	lines.Lines.Index = 1
	lines.Lines.Lines[0].Selected = true
	/*/
	lines := NewTcellLines()

	// XXX set settings...
	// XXX

	//*/

	os.Exit(
		toExitCode(
			lines.Loop())) }



// vim:set sw=4 ts=4 nowrap :
