
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
	// XXX
}
func (this *TcellDrawer) Setup() *TcellDrawer {
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Panic(err) }
	this.Screen = screen
	// XXX
	return this }
func (this *TcellDrawer) drawCells(col, row int, str string, style string) {
	// XXX get the style...
	// XXX
	//for i, r := range []rune(str) {
	//	// XXX draw cell
	//}
}

func main(){
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

	fmt.Printf("--- %#v\n", lines)
}



// vim:set sw=4 ts=4 nowrap :
