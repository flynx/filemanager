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

	buf := bytes.Buffer{}
	done_input := make(chan bool)
	go func(){
		var copying sync.Mutex
		scanner := bufio.NewScanner(ir)
		for scanner.Scan() {
			txt := scanner.Text()
			fmt.Println(">>>", txt)
			io.WriteString(&buf, txt +"\n")
			if copying.TryLock() {
				go func(){
					io.Copy(ow, &buf)
					copying.Unlock() }() } }
		copying.Lock()
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
