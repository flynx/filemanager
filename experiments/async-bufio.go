package main

import (
	"fmt"
	"bufio"
	//"bytes"
	"io"
	"time"
	"sync"
)


// XXX take either Reader/WriteCloser or Reader/Writer...
// XXX do we need to control this in any way???
func PipeTee(reader io.Reader, writer io.WriteCloser, handler func(string)) {
	writer_buf := bufio.NewWriter(writer)
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
	// finalize things...
	output.Lock()
	writer_buf.Flush()
	// XXX only do this if WriteCloser...
	writer.Close() }


// XXX make this a generic Async(func, ...args)
func AsyncPipeTee(reader io.Reader, writer io.WriteCloser, handler func(string)) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		PipeTee(reader, writer, handler)
		close(done) }()
	return done }





func main(){

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	done_input := AsyncPipeTee(ir, ow,
		func(s string){
			fmt.Println(">>>", s) })


	// output chain...
	done_output := make(chan bool)
	go func(){
		scanner := bufio.NewScanner(or)
		for scanner.Scan() {
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
