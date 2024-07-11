package main

import (
	"fmt"
	"bufio"
	"bytes"
	"io"
	"time"
	"sync"
)


func main(){

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	// XXX still truncating the tail sometimes...
	done_input := make(chan bool)
	go func(){
		buf := bytes.Buffer{}
		prebuf := bytes.Buffer{}
		var copying sync.Mutex
		scanner := bufio.NewScanner(ir)
		for scanner.Scan() {
			txt := scanner.Text()
			fmt.Println(">>>", txt)
			// nothing is in queue to pipe -- flush pre-buffer, write...
			if copying.TryLock() {
				//fmt.Println("  (flush/buf)", txt)
				buf.ReadFrom(&prebuf)
				buf.WriteString(txt +"\n")
				// write to output...
				go func(){
					defer copying.Unlock() 
					//fmt.Println("  (flush: buf)")
					io.Copy(ow, &buf) }() 
			// waiting to write -- pre-buffer...
			} else {
				//fmt.Println("  (prebuf)", txt)
				io.WriteString(&prebuf, txt +"\n") } }
		copying.Lock()
		// flush the pre-buffer...
		if prebuf.Len() > 0 {
			//fmt.Println("  (flush: prebuf)")
			io.Copy(ow, &prebuf) }
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

	io.WriteString(iw, "A\n")
	io.WriteString(iw, "B\n")
	time.Sleep(time.Millisecond*100)
	io.WriteString(iw, "C\n")
	iw.Close()


	<-done_input
	<-done_output

}
