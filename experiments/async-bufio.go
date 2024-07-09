package main

import (
	"fmt"
	"bufio"
	//"bytes"
	"io"
	"time"
	"sync"
)


//
//	reader -> tee -> handler(line)
//				\
//				 +-> writer
//
// NOTE: the handler is called sync, this if it blocks it will block the 
//		line write to writer
//		XXX should this be the case???
// NOTE: this buffers the writes and will not block on the writer
func Tee(reader io.Reader, writer io.Writer, handler func(string)) {
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
	writer_buf.Flush() }


// XXX make this a generic Async(func, ...args)
func AsyncTee(reader io.Reader, writer io.Writer, handler func(string)) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		Tee(reader, writer, handler)
		close(done) }()
	return done }

// XXX make this a generic Async(func, ...args)
func AsyncTeeCloser(reader io.Reader, writer io.WriteCloser, handler func(string)) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		Tee(reader, writer, handler)
		writer.Close() 
		close(done) }()
	return done }





func main(){

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	// XXX this does not yet work without a reader...
	done_input := AsyncTeeCloser(ir, ow,
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
	//*/

	io.WriteString(iw, "A\n")
	io.WriteString(iw, "B\n")
	time.Sleep(time.Millisecond*100)
	io.WriteString(iw, "C\n")
	iw.Close()


	<-done_input
	<-done_output

}
