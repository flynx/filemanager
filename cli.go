
package main

import (
	//"fmt"
	"log"
	"os"
	//"slices"
	//"reflect"

	"github.com/gdamore/tcell/v2"
	//"github.com/jessevdk/go-flags"

	//"github.com/mkideal/cli"
)


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
func (this *TcellDrawer) Update() *TcellDrawer {

	// XXX update geometry...

	return this }
func (this *TcellDrawer) Loop() Result {
	defer this.Finalize()
	// XXX event loop ???
	for {
		this.Update()
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

func (this *TcellDrawer) drawCells(col, row int, str string, style string) {
	if style == "EOL" {
		return }
	// XXX get the style...
	// XXX STUB
	s := tcell.StyleDefault
	for i, r := range []rune(str) {
		this.SetContent(col+i, row, r, nil, s) } }

func NewTcellLines(l ...Lines) TcellDrawer {
	var lines Lines
	if len(l) == 0 {
		lines = Lines{}
	} else {
		lines = l[0] }

	// XXX  should this be the controller???
	//		...since this would house the event look it would be logical...
	//		on the other hand referencing to lines here would introduce 
	//		a circular reference (does not feel good)...
	drawer := TcellDrawer{}

	// XXX can we combine drawer and lines and not jugle two things???
	//		- we could add .Setup(..) to CellsDrawer inteface but this
	//			will only partially solve the issue -- needing the ref
	//			for other methods/stuff...
	//		- a different approach to extension???
	drawer.Setup(lines)

	return drawer }




func main(){
	//* XXX stub...
	lines := NewTcellLines(Lines{
		//SpanMode: "*,5",
		Width: 20,
		Height: 6,
		Border: "│┌─┐│└─┘",
	})
	lines.Lines.Write("Some text")
	/*/
	lines := NewTcellLines()
	//*/

	// XXX set settings...


	os.Exit(
		toExitCode(
			lines.Loop())) }



// vim:set sw=4 ts=4 nowrap :
