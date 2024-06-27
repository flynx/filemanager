
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



// XXX this can sometimes truncate output -- sync error??
func TestRaw(t *testing.T){
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

// XXX this can sometimes truncate output -- sync error??
func TestBasics(t *testing.T){
	c := Cmd{}
	//cmd := "pwd; sleep 1; ls"
	cmd := "pwd; sleep 0.2; ls"

	fmt.Println("$", cmd)
	if _, err := c.Run(cmd, nil); err != nil {
		t.Fatal(err) }

	go func(){
		scanner := bufio.NewScanner(c.Stdout)
		for scanner.Scan() {
			fmt.Println("    >>", scanner.Text()) } }()

	//fmt.Println("async")
	<-c.Done
	fmt.Println("done.")
}

// XXX this can sometimes truncate output -- sync error??
func TestRun(t *testing.T){
	cmd, in, err := RunFilter(
		//"cat", 
		"grep moo", 
		func(line string){
			fmt.Println("    >>", line) })
	if err != nil {
		t.Fatal(err) }

	//time.Sleep(time.Second)
	io.WriteString(in, "foo\n")
	cmd.WriteString("moo\n")
	time.Sleep(time.Second)
	cmd.WriteString("boo\n")
	cmd.WriteString("moo\n")

	fmt.Println("async")
	in.Close()

	<-cmd.Done
}


