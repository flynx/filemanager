package main

import (
	"fmt"
	"bufio"
	//"bytes"
	"io"
	"time"
	//"sync"
)



func main(){

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	reader_buf := bufio.NewReader(or)
	done_input := make(chan bool)
	go func(){
		scanner := bufio.NewScanner(ir)
		for scanner.Scan() {
			txt := scanner.Text()
			fmt.Println(">>>", txt)
			// XXX for some reason the reader_buf does not read things 
			//		from the pipe untill requested -- if we remove the 
			//		scanner loop below this will block on first write...
			io.WriteString(ow, txt +"\n") 
		}
		close(done_input) }()


	// output chain...
	done_output := make(chan bool)
	go func(){
		time.Sleep(time.Millisecond*2000)
		scanner := bufio.NewScanner(reader_buf)
		for scanner.Scan() {
			time.Sleep(time.Millisecond*100)
			txt := scanner.Text()
			fmt.Println("  >", txt) }
		close(done_output) }()

	io.WriteString(iw, "A\n")
	io.WriteString(iw, "B\n")
	time.Sleep(time.Millisecond*100)
	io.WriteString(iw, "C\n")
	iw.Close()


	<-done_input
	<-done_output

}
