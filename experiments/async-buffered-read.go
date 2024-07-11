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
				io.Copy(&buf, &prebuf)
				io.WriteString(&buf, txt +"\n") 
				copying.Unlock()
			// waiting to write -- pre-buffer...
			} else {
				io.WriteString(&prebuf, txt +"\n") } 
			// write to pipe...
			if copying.TryLock() {
				go func(){
					defer copying.Unlock() 
					io.Copy(ow, &buf) }() } }
		copying.Lock()
		// flush the pre-buffer...
		if prebuf.Len() > 0 {
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
	//<-done_output

}
