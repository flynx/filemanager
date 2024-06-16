
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
	r := Cmd{}
	//cmd := "pwd; sleep 1; ls"
	cmd := "pwd; sleep 0.2; ls"

	fmt.Println("$", cmd)
	r.Run(cmd, nil)

	go func(){
		scanner := bufio.NewScanner(r.Stdout)
		for scanner.Scan() {
			fmt.Println("    >>", scanner.Text()) } }()

	//fmt.Println("async")
	<-r.Done
	fmt.Println("done.")
}


func TestRun(t *testing.T){
	/* XXX
	out, in := io.Pipe()
	cmd, _ := Run("cat", out)
	cmd.HandleLine(
		func(line string){
			fmt.Println("    >>", line) })
	/*/
	cmd, in, _ := RunFilter(
		"cat", 
		func(line string){
			fmt.Println("    >>", line) })
	//*/

	//time.Sleep(time.Second)
	io.WriteString(in, "moo\n")
	io.WriteString(in, "foo\n")
	time.Sleep(time.Second)
	io.WriteString(in, "boo\n")
	io.WriteString(in, "moo\n")

	fmt.Println("async")
	in.Close()

	<-cmd.Done
}

func TestRawMIMO(t *testing.T){
	/*/ XXX
	cmd := "cat"
	c := exec.Command(cmd)
	/*/
	cmd := "bash -c 'cat'"
	c := exec.Command("bash", "-c", "cat")
	//*/

	// XXX both of these work...
	o, in := io.Pipe()
	c.Stdin = o
	/*/
	in, _ := c.StdinPipe()
	//*/
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

