package main

import (
	"fmt"
	"bufio"
	//"bytes"
	"io"
	"time"
	"sync"
)


// XXX do we need to control this in any way???
func PipeTee(reader io.ReadCloser, writer io.WriteCloser, handler func(string)) (<-chan bool) {
	writer_buf := bufio.NewWriter(writer)
	done_input := make(chan bool)
	go func(){
		scanner := bufio.NewScanner(reader)
		var output sync.Mutex
		for scanner.Scan() {
			txt := scanner.Text()
			if handler != nil {
				handler(txt) }
			// NOTE this needs to be non-blocking this is still blocking...
			// NOTE seems that bufio is not thread-safe as not explicitly
			//		sychronising writing and flushing produces unpredictable 
			//		putput (races)
			output.Lock()
			io.WriteString(writer_buf, txt +"\n") 
			output.Unlock()
			// keep only one blocked .Flush()...
			if output.TryLock() {
				go func(){ 
					writer_buf.Flush() 
					output.Unlock() }() } }
		output.Lock()
		writer_buf.Flush()
		writer.Close()
		close(done_input) }() 
	return done_input }


func main(){

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	done_input := PipeTee(ir, ow, 
		func(s string){
			fmt.Println(">>>", s) })


	// output chain...
	done_output := make(chan bool)
	go func(){
		scanner := bufio.NewScanner(or)
		for scanner.Scan() {
			//time.Sleep(time.Millisecond*200)
			txt := scanner.Text()
			fmt.Println("  >", txt) }
		close(done_output) }()

	io.WriteString(iw, "A\n")
	io.WriteString(iw, "B\n")
	time.Sleep(time.Millisecond*50)
	io.WriteString(iw, "C\n")
	iw.Close()


	<-done_input
	<-done_output

	//time.Sleep(time.Second)

}
