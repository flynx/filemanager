
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

func TestRawPipe(t *testing.T){

	done := make(chan bool)

	// XXX for some reason grep waits for pipe to close here but cat does not...
	filter := exec.Command("bash", "-c", "grep go")
	//filter := exec.Command("bash", "-c", "cat")
	in, _ := filter.StdinPipe()
	out, _ := filter.StdoutPipe()
	go func(){
		defer close(done) 
		defer out.Close()
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			fmt.Println("  out:", scanner.Text()) } }()
	filter.Start()

	source := exec.Command("bash", "-c", "ls")
	src, _ := source.StdoutPipe()
	go func(){
		defer in.Close() 
		scanner := bufio.NewScanner(src)
		for scanner.Scan() {
			time.Sleep(time.Millisecond*50)
			line := scanner.Text()
			fmt.Println("src:", line)
			io.WriteString(in, line +"\n") } }()
	source.Start()

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
	cmd, err := RunFilter(
		//"cat", 
		"grep moo", 
		func(line string){
			fmt.Println("    >>", line) })
	if err != nil {
		t.Fatal(err) }

	//time.Sleep(time.Second)
	io.WriteString(cmd.Stdin, "foo\n")
	cmd.WriteString("moo\n")
	time.Sleep(time.Second)
	cmd.WriteString("boo\n")
	cmd.WriteString("moo\n")

	fmt.Println("async")
	cmd.Stdin.Close()

	<-cmd.Done
}

func TestFilter(t *testing.T){
	filter, err := RunFilter(
		// XXX MAGIC: cat works, grep does not for some reason...
		//"sed 's/go/GO/g'",
		"grep go", 
		//"cat", 
		func(line string){
			fmt.Println("    <<", line) })
	if err != nil {
		t.Fatal(err) }

	runner, _ := Run("ls", nil) 
	scanner := bufio.NewScanner(runner.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("    >>", line)
		filter.WriteString(line +"\n") }

	filter.WriteString("moo\n")
	filter.WriteString("go\n")

	time.Sleep(time.Second)

	filter.Stdin.Close()

	fmt.Println("done.")
}


