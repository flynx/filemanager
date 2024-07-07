
package main

import (
	"testing"
	"fmt"
	"strings"
	"os"
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
		func(s string){
			fmt.Println("  grep:", s)
			n++ })
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

	ls.Wait()
	grep.Stdin.Close()
	grep.Wait()

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}

/* XXX still need to figure this out...
func TestPipe(t *testing.T) {
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
		// XXX ERR: this will consume .Stdout...
		func(s string){ 
			fmt.Println("ls:", s) })
	if err != nil {
		t.Error(err) }

	// grep...
	var grep *PipedCmd
	grep, err = ls.PipeTo("grep '.go'", 
		func(s string){
			fmt.Println("  grep:", s)
			n++ })

	ls.Wait()
	grep.Stdin.Close()
	grep.Wait()

	if c != n {
		t.Errorf("Skipped part of grep output, expected: %v got: %v\n", c, n) } 
}
//*/


