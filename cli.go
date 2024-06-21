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
	"strconv"
	"slices"
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

	// Geometry
	//
	// Format:
	//		"auto" | "50%" | "20"
	Width string
	Height string

	// Format:
	//
	Top string
	Left string

	// Format:
	//		
	Align []string


	// caches...
	__style_cache map[string]tcell.Style
	__int_cache map[string]int
	__float_cache map[string]float64
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

func (this *TcellDrawer) updateGeometry() *TcellDrawer {
	var err error
	W, H := this.Screen.Size()
	Width, Height := this.Width, this.Height
	Align := this.Align
	if len(Align) == 0 {
		Align = []string{"top", "left"} }

	// XXX should this be more generic???
	// XXX revise the error case...
	cachedFloat := func(str string) float64 {
		v, ok := this.__float_cache[str]
		if ! ok {
			var err error
			// handle "%"...
			if str[len(str)-1] == '%' {
				str = string(str[0:len(str)-1]) }
			v, err = strconv.ParseFloat(str, 32)
			if err != nil {
				log.Println(err) } 
			this.__float_cache[str] = v }
		return v }

	// Width...
	if Width == "auto" || 
			Width == "" {
		this.Lines.Width = W
	} else if Width[len(Width)-1] == '%' {
		this.Lines.Width = int(float64(W) * (cachedFloat(Width) / 100))
	} else {
		// XXX revise the error case + cache???
		this.Lines.Width, err = strconv.Atoi(Width)
		if err != nil {
			log.Println("Error parsing width", Width) } }
	// Height...
	if Height == "auto" || 
			Height == "" {
		this.Lines.Height = H
	} else if Height[len(Height)-1] == '%' {
		this.Lines.Height = int(float64(H) * (cachedFloat(Height) / 100))
	} else {
		// XXX revise the error case + cache???
		this.Lines.Height, err = strconv.Atoi(Height)
		if err != nil {
			log.Println("Error parsing height", Height) } }

	// Left (value)
	left_set := false
	if slices.Contains(Align, "left") {
		left_set = false
		this.Lines.Left = 0
	} else if slices.Contains(Align, "right") {
		left_set = false
		this.Lines.Left = W - this.Lines.Width
	} else if Align[0] != "center" {
		left_set = false
		// XXX revise the error case + cache???
		this.Lines.Left, err = strconv.Atoi(Align[0])
		if err != nil {
			log.Println("Error parsing left", Align[0]) } }
	// Top (value)
	top_set := false
	if slices.Contains(Align, "top") {
		top_set = false
		this.Lines.Top = 0
	} else if slices.Contains(Align, "bottom") {
		top_set = false
		this.Lines.Top = H - this.Lines.Height
	} else if Align[1] != "center" {
		top_set = false
		// XXX revise the error case + cache???
		this.Lines.Top, err = strconv.Atoi(Align[1]) 
		if err != nil {
			log.Println("Error parsing top", Align[1]) } }
	// Left (center)
	if ! left_set {
		if top_set && 
				slices.Contains(Align, "center") || 
				Align[0] == "center" {
			this.Lines.Left = int(float64(W - this.Lines.Width) / 2) } }
	// Top (center)
	if ! top_set {
		if top_set && 
				slices.Contains(Align, "center") || 
				Align[0] == "center" {
			this.Lines.Top = int(float64(H - this.Lines.Height) / 2) } }
	return this }
func (this *TcellDrawer) handleScrollLimits() *TcellDrawer {
	// XXX
	return this}

func (this *TcellDrawer) Loop() Result {
	defer this.Finalize()
	for {
		this.updateGeometry()
		this.Lines.Draw()
		this.Show()

		evt := this.PollEvent()

		switch evt := evt.(type) {
			// keep the selection in the same spot...
			case *tcell.EventResize:
				// XXX we do not need to .updateGeometry() as we are doing it above... 
				//this.updateGeometry()
				this.handleScrollLimits()
			// XXX STUB exit on keypress...
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

// XXX might be nice to be able to set flags like underline, bold, italic, ...etc.
// XXX BUG: "background"/"foreground" do not work as we can't yet get 
// 		the actual default colors...
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
	// XXX this returns "default" "default" -- very usefull...
	bg, fg, _ := tcell.StyleDefault.Decompose()
	if style[0] != "default" && 
			style[0] != "foreground" && 
			style[0] != "fg" {
		switch style[0] {
			case "background":
				log.Println("Style2TcellStyle(..): \"background\" / \"foreground\" colors do not work yet.")
				res = res.Foreground(bg)
			default:
				res = res.Foreground(tcell.GetColor(style[0])) } }
	if style[1] != "default" &&
			style[1] != "background" && 
			style[1] != "bg" {
		switch style[1] {
			case "foreground":
				log.Println("Style2TcellStyle(..): \"background\" / \"foreground\" colors do not work yet.")
				res = res.Background(fg)
			default:
				res = res.Background(tcell.GetColor(style[1])) } }
	return res }
func (this *TcellDrawer) drawCells(col, row int, str string, style_name string, style Style) {
	if style_name == "EOL" {
		return }
	// get style (cached)...
	if this.__style_cache == nil {
		this.__style_cache = map[string]tcell.Style{} }
	s, ok := this.__style_cache[style_name]
	if ! ok {
		s := Style2TcellStyle(style)
		this.__style_cache[style_name] = s }
	// draw...
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
		SpanSeparator: "│",
		Border: "│┌─┐│└─┘",
	})
	lines.Lines.Append(
		"Some text",
		"Current%SPAN",
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
