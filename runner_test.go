
package main

import (
	"testing"
	//"bytes"
	"io"
	"bufio"
	//"strconv"
	"fmt"
	"os/exec"
	"time"
)


func TestBasics(t *testing.T){
	r := Runner{}
	//cmd := "pwd; sleep 1; ls"
	cmd := "pwd; sleep 0.2; ls"

	fmt.Println("$", cmd)
	//r.Run(cmd, &bytes.Buffer{})
	r.Run(cmd)

	go func(){
		scanner := bufio.NewScanner(r.Stdout)
		for scanner.Scan() {
			fmt.Println("    >>", scanner.Text()) } }()

	//fmt.Println("async")
	<-r.Done
	fmt.Println("done.")
}

func TestRawMIMO(t *testing.T){
	// XXX
	cmd := "cat"
	c := exec.Command(cmd)
	/*/
	cmd := "bash -c 'cat'"
	c := exec.Command("bash", "-c", "cat")
	//*/

	in, _ := c.StdinPipe()
	out, _ := c.StdoutPipe()

	fmt.Println(cmd)

	c.Start()

	done := make(chan bool)

	go func(){
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			fmt.Println("    >>", scanner.Text()) } }()
	go func(){
		c.Wait()
		close(done) }()

	io.WriteString(in, "moo\n")
	io.WriteString(in, "foo\n")
	time.Sleep(time.Second)
	io.WriteString(in, "boo\n")
	io.WriteString(in, "moo\n")

	fmt.Println("async")
	in.Close()
	//time.Sleep(time.Second)

	//fmt.Println("async")
	<-done
	fmt.Println("done.")
}

