
package main

import (
	"testing"
	"fmt"
	"strings"
	"os"
	"io"
	"bufio"
	"time"
)



var RunCount = 1000
func TestRun(t *testing.T) {
	//SHELL = ""
	//PREFIX = ""

	files, _ := os.ReadDir(".")
	c := len(files) + 2
	s := 0
	report := func(n int) {
		if n != c {
			fmt.Printf("\nListed: %v of %v (no errors)\n", n, c)
			s++
		} else {
			fmt.Print(".") } }

	for i := 0; i < RunCount; i++ {
		n := 0
		cmd, err := Run("ls -a", func(s string){ n++ })
		if err != nil {
			t.Error(err) }
		if err := cmd.Wait(); err != nil {
			t.Error(err) }
		report(n) }
	fmt.Println("")

	if s > 0 {
		t.Errorf("Skipped part of the output %v times of %v\n", s, RunCount) } 
}


func TestTeeBasic(t *testing.T) {
	r1, w1 := io.Pipe()
	_, w2 := io.Pipe()

	done := AsyncTeeCloser(r1, w2, 
		func(s string) bool {
			time.Sleep(time.Millisecond*5)
			fmt.Println(">>>", s) 
			return true })

	time.Sleep(time.Millisecond*10)
	io.WriteString(w1, "A\n")
	io.WriteString(w1, "B\n")
	io.WriteString(w1, "C\n")
	io.WriteString(w1, "D\n")
	w1.Close()

	<-done
}
func TestTeeChain(t *testing.T) {

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	r3, w3 := io.Pipe()
	r4, w4 := io.Pipe()

	go TeeCloser(r1, w2, 
		func(s string) bool {
			time.Sleep(time.Millisecond*5)
			fmt.Println(">>>", s) 
			return true })

	go TeeCloser(r2, w3, nil)

	go TeeCloser(r3, w4, 
		func(s string) bool {
			time.Sleep(time.Millisecond*10)
			fmt.Println(" >>", s) 
			return true })

	done := AsyncTeeCloser(r4, nil, 
		func(s string) bool {
			fmt.Println("  >", s) 
			return true })

	time.Sleep(time.Millisecond*10)
	io.WriteString(w1, "A\n")
	io.WriteString(w1, "B\n")
	io.WriteString(w1, "C\n")
	io.WriteString(w1, "D\n")
	w1.Close()

	<-done
}

func TestPipeManual(t *testing.T) {
	n := 0
	c := 0
	files, _ := os.ReadDir(".")
	for _, e := range files {
		if strings.Contains(e.Name(), ".go") {
			c++ } }

	var err error

	// grep...
	var grep *PipedCmd
	grep, err = Pipe("grep '.go'", 
		func(s string) bool {
			fmt.Println("  grep:", s)
			n++ 
			return true })
	if err != nil {
		t.Error(err) }

	// ls...
	var ls *Cmd
	ls, err = Run("ls -a", 
		func(s string){ 
			fmt.Println("ls:", s)
			grep.Writeln(s) })
	if err != nil {
		t.Error(err) }

	go func(){
		ls.Wait()
		// NOTE: sice we are manually connecting pipes we also need to 
		//		manually close them...
		grep.Close() }()

	grep.Wait()

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}

func TestCmd(t *testing.T) {
	files, _ := os.ReadDir(".")
	// NOTE: the +2 here is to account for "." and ".." returned by ls -a
	c := len(files) + 2
	n := 0

	var err error

	// ls...
	var ls *Cmd
	ls, err = Run("ls -a", 
		func(s string){ 
			fmt.Println("ls:", s) 
			n++ })
	if err != nil {
		t.Error(err) }


	ls.Wait()

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}
func TestCmdStdout(t *testing.T) {
	n := 0
	c := 0
	var err error

	// ls...
	var ls *Cmd
	ls, err = Run("ls -a", 
		func(s string){ 
			fmt.Println("ls:", s) 
			c++ })
	if err != nil {
		t.Error(err) }

	scanner := bufio.NewScanner(ls.Stdout)
	for scanner.Scan() {
		fmt.Println("  ->", scanner.Text()) 
		n++ }

	ls.Wait()

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}

func TestPipeChainPassive(t *testing.T) {
	n := 0
	c := 0
	files, _ := os.ReadDir(".")
	for _, e := range files {
		if strings.Contains(e.Name(), ".go") {
			c++ } }

	var err error

	// ls...
	var ls *Cmd
	ls, err = Run("ls -a")
	if err != nil {
		t.Error(err) }

	// grep...
	var grep *PipedCmd
	grep, err = ls.PipeTo("grep '.go'", 
		func(s string) bool {
			fmt.Println("  grep:", s)
			n++ 
			return true })

	grep.Wait()

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}
func TestPipeChainActive(t *testing.T) {
	n := 0
	c := 0
	files, _ := os.ReadDir(".")
	for _, e := range files {
		if strings.Contains(e.Name(), ".go") {
			c++ } }

	var err error

	// ls...
	var ls *Cmd
	ls, err = Run("ls -a", 
		func(s string){ 
			time.Sleep(time.Millisecond*50)
			fmt.Println("ls:", s) })
	if err != nil {
		t.Error(err) }

	// grep...
	var grep *PipedCmd
	grep, err = ls.PipeTo("grep '.go'", 
		func(s string) bool {
			fmt.Println("  grep:", s)
			n++ 
			return true })

	grep.Wait()

	//time.Sleep(time.Millisecond*500)

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}


