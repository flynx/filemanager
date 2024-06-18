
package main

import (
	"fmt"
	//"log"
	//"os"
	//"slices"
	//"reflect"

	//"github.com/gdamore/tcell/v2"
	//"github.com/jessevdk/go-flags"

	//"github.com/mkideal/cli"
)


type TcellDrawer struct {
}
func (this *TcellDrawer) drawCells(col, row int, str string, style string) {
	// XXX
}

func main(){
	lines := Lines{
		CellsDrawer: &TcellDrawer{},
	}

	fmt.Printf("--- %#v\n", lines)
}



// vim:set sw=4 ts=4 nowrap :
