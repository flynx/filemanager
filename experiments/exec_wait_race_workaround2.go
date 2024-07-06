
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"io"
	//"time"
	"strings"
	"errors"
)

var SHELL = "bash -c"
var PREFIX = "stdbuf -i0 -oL -eL"

type LineHandler func(string)
type Cmd struct {
	*exec.Cmd

	Shell string
	Prefix string
	Code string

	Handler LineHandler

	Stdin io.WriteCloser
	Stdout io.Reader
	Stderr io.Reader

	Done <- chan bool
}
func (this *Cmd) Run(stdin ...io.Reader) error {
	if this.Cmd != nil {
		return errors.New(".Run(..): previous command not done.") }

	done := make(chan bool)
	this.Done = done

	shell := this.Shell
	if shell == "" {
		shell = SHELL }
	prefix := this.Prefix
	if prefix == "" {
		prefix = PREFIX }

	// setup the command...
	var cmd *exec.Cmd
	if shell == "" {
		s := strings.Fields(prefix +" "+ this.Code)
		cmd = exec.Command(s[0], s[1:]...)
	} else {
		s := strings.Fields(shell)
		cmd = exec.Command(s[0], append(s[1:], prefix +" "+ this.Code)...) }
	this.Cmd = cmd

	// i/o...
	// XXX revise...
	if len(stdin) > 0 {
		this.Cmd.Stdin = stdin[0] }
	if p, err := cmd.StdoutPipe(); err == nil {
		this.Stdout = p
	} else {
		return err }
	if p, err := cmd.StderrPipe(); err == nil {
		this.Stderr = p
	} else {
		return err }

	go func(){
		scanner := bufio.NewScanner(this.Stdout)
		for scanner.Scan() {
			this.Handler(scanner.Text()) }
		this.Cmd.Wait()
		this.Cmd = nil 
		this.Stdin = nil
		close(done) }()

	return this.Cmd.Start() }
func (this *Cmd) Wait() error {
	<-this.Done
	return nil }

// XXX revise name...
func (this *Cmd) Write(s string) (int, error) {
	if this.Stdin == nil {
		return 0, errors.New(fmt.Sprint(".Write(..): can not write to .Stdin:", this.Stdin)) }
	return io.WriteString(this.Stdin, s) }
func (this *Cmd) Writeln(s string) (int, error) {
	return this.Write(s +"\n") }


func Run(code string, handler LineHandler) (*Cmd, error) {
	this := Cmd{}
	this.Code = code
	this.Handler = handler
	if err := this.Run(); err != nil {
		return &this, err }
	return &this, nil }

func Pipe(code string, handler LineHandler) (*Cmd, error) {
	this := Cmd{}
	this.Code = code
	this.Handler = handler
	r, w := io.Pipe()
	this.Stdin = w
	if err := this.Run(r); err != nil {
		return &this, err }
	go func(){
		this.Wait()
		w.Close() }()
	return &this, nil }



var RunCount = 100
func main() {

	// XXX make this comparable to ./exec_wait_race_workaround.go in 
	//		speed -- as both of these add a significant overhead...
	SHELL = ""
	PREFIX = ""

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
		cmd, _ := Run("ls -a", func(s string){ n++ })
		cmd.Wait()
		report(n) }
	fmt.Println("")

	if s > 0 {
		fmt.Printf("Skipped part of the output %v times of %v\n", s, RunCount) } }



