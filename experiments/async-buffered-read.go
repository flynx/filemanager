package main

import (
	"fmt"
	"bufio"
	"bytes"
	"io"
	"time"
	"sync"
)


func Tee(reader io.Reader, writer io.Writer, handler func(string)) {
	buf := bytes.Buffer{}
	prebuf := bytes.Buffer{}
	var copying sync.Mutex
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		txt := scanner.Text()
		if handler != nil {
			handler(txt) }
		// nothing is in queue to pipe -- flush pre-buffer, write...
		if copying.TryLock() {
			buf.ReadFrom(&prebuf)
			buf.WriteString(txt +"\n")
			// write to output...
			go func(){
				defer copying.Unlock() 
				io.Copy(writer, &buf) }() 
		// waiting to write -- pre-buffer...
		} else {
			io.WriteString(&prebuf, txt +"\n") } }
	copying.Lock()
	// flush the pre-buffer if non-empty...
	if prebuf.Len() > 0 {
		io.Copy(writer, &prebuf) } }



func main(){

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	done_input := make(chan bool)
	go func(){
		Tee(ir, ow,
			func(s string){
				fmt.Println(">>>", s) })
		ow.Close()
		close(done_input) }()


	// output chain...
	done_output := make(chan bool)
	go func(){
		//time.Sleep(time.Millisecond*2000)
		scanner := bufio.NewScanner(or)
		for scanner.Scan() {
			time.Sleep(time.Millisecond*100)
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



	ir, iw = io.Pipe()
	buf := bytes.Buffer{}
	done_input = make(chan bool)
	go func(){
		Tee(ir, &buf,
			func(s string){
				fmt.Println(">>>", s) })
		ow.Close()
		close(done_input) }()


	io.WriteString(iw, "A\n")
	io.WriteString(iw, "B\n")
	time.Sleep(time.Millisecond*100)
	io.WriteString(iw, "C\n")
	iw.Close()


	<-done_input


}
