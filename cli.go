
package main

import (
	"fmt"
	"log"
	//"os"
	//"slices"
	//"reflect"

	"github.com/gdamore/tcell/v2"
	//"github.com/jessevdk/go-flags"

	//"github.com/mkideal/cli"
)


type TcellDrawer struct {
	tcell.Screen

	Lines Lines
	// XXX
}
func (this *TcellDrawer) Setup() *TcellDrawer {
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
func (this *TcellDrawer) Loop() *TcellDrawer {
	/*/ XXX event loop ???
	for {
		this.Update()
		// XXX draw...
		this.Show()

		evt := screen.PollEvent()

		switch evt := evt.(type) {
			// XXX
		} }
	//*/
	return this }
// handle panics and cleanup...
func (this *TcellDrawer) Finalize() *TcellDrawer {
	maybePanic := recover()
	this.Screen.Fini()
	if maybePanic != nil {
		panic(maybePanic) } 
	return this }

func (this *TcellDrawer) drawCells(col, row int, str string, style string) {
	// XXX get the style...
	// XXX
	/* XXX
	for i, r := range []rune(str) {
		// XXX draw cell
	}
	//*/
}




func main(){
	// XXX  should this be the controller???
	//		...since this would house the event look it would be logical...
	//		on the other hand referencing to lines here would introduce 
	//		a circular reference (does not feel good)...
	drawer := TcellDrawer{
		// XXX
	}
	lines := Lines{
		CellsDrawer: &drawer,
		// XXX
	}

	// XXX can we combine drawer and lines and not jugle two things???
	//		- we could add .Setup(..) to CellsDrawer inteface but this
	//			will only partially solve the issue -- needing the ref
	//			for other methods/stuff...
	//		- a different approach to extension???
	drawer.Setup()
	defer drawer.Finalize()

	// XXX start the event loop...
	//		...should this be in os.Exit(..) ???
	drawer.Loop()

	fmt.Printf("--- %#v\n", lines)
}



// vim:set sw=4 ts=4 nowrap :
