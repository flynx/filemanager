/*
*	
*
*/

package main

import (
	"fmt"
	"log"
	"io"
	"os/exec"
	"strings"
	//"strconv"
	//"slices"
	"bufio"
	//"bytes"
	//"sync"
	//"regexp"
	//"os"
)


var SHELL = "bash -c"
var PREFIX = "stdbuf -i0 -oL -eL"

type Cmd struct {
	*exec.Cmd

	Shell string
	Prefix string
	Code string

	State string
	Error error
	Done <-chan bool

	// XXX is overloading this a good idea???
	//		...these face the oposite direction relative to exec.Cmd...
	Stdin io.WriteCloser

	Stdout io.Reader
	Stderr io.Reader

	LineHandler LineHandler
}

// Sorthands...
//
// XXX add ability to auto restart without losing context...
func Run(code string, stdin io.Reader) (*Cmd, error) {
	cmd := Cmd{
		Code: code,
	}
	if c, err := cmd.Run(code, stdin) ; err != nil {
		return c, err }
	return &cmd, nil }


func Piped(code string) (*Cmd, error) {
	cmd := Cmd{
		Code: code,
	}
	r, w := io.Pipe()
	if c, err := cmd.Run(code, r) ; err != nil {
		return c, err }
	cmd.Stdin = w
	return &cmd, nil }


// XXX do we need this...
func RunFilter(code string, handler LineHandler) (*Cmd, error) {
	// XXX error handling makes this code quite ugly, is there a clearer way to do this???
	cmd, err := Piped(code)
	if err != nil {
		return cmd, err }
	if cmd, err = cmd.HandleLine(handler); err != nil {
		return cmd, err }
	return cmd, nil }


// XXX list of function???
type LineHandler func(string)
func (this *Cmd) HandleLine(handler LineHandler) (*Cmd, error) {
	if this.LineHandler != nil {
		return this, fmt.Errorf(".HandleLine(..): can not assign multiple handlers.") }
	this.LineHandler = handler
	go func(){
		scanner := bufio.NewScanner(this.Stdout)
		for scanner.Scan() {
			handler(scanner.Text()) } }() 
	return this, nil }
// XXX this is implemented in lines2.go...
func (this *Cmd) makeEnv(){
	// XXX
}
// XXX make this restartable... (???)
//		...for this to work ew'll need to also handle stdin/stdout/stderr
//		correctly... not sure how to do this when they are closed...
// XXX should this be public???
func (this *Cmd) Run(code string, stdin io.Reader) (*Cmd, error) {
	// can't run twice...
	if this.State != "" {
		return this, fmt.Errorf(".Run(..): can not run a command a %v command.", this.State) }
	shell := this.Shell
	if shell == "" {
		shell = SHELL }
	prefix := this.Prefix
	if prefix == "" {
		prefix = PREFIX }
	s := strings.Fields(shell)
	// setup the command...
	cmd := exec.Command(s[0], append(s[1:], prefix +" "+ code)...)
	this.Cmd = cmd
	//cmd.Env = this.makeEnv()
	this.State = "ready"

	if stdin != nil {
		cmd.Stdin = stdin }
	var err error
	this.Stdout, err = cmd.StdoutPipe()
	if err != nil {
		return this, err }
	this.Stderr, err = cmd.StderrPipe()
	if err != nil {
		return this, err }

	done := make(chan bool)
	this.Done = done
	// set state...
	go func(){
		res := <-done
		if res {
			this.State = "done"
		} else {
			this.State = "failed" } 
		close(done) }()

	// run...
	if err := cmd.Start(); err != nil {
		return this, err }
	this.State = "running"

	// cleanup...
	go func(){
		done_state := true
		// handle errors...
		if err := cmd.Wait(); err != nil {
			log.Println("Error executing: \""+ code +"\"", err) 
			scanner := bufio.NewScanner(this.Stderr)
			lines := []string{}
			for scanner.Scan() {
				lines = append(lines, scanner.Text()) }
			log.Println("    ERR:", strings.Join(lines, "\n"))
			//log.Println("    ENV:", env)
			this.Error = err
			done_state = false }
		// XXX do we invalidate .Cmd ???
		//this.Cmd = nil
		done <- done_state }()

	return this, nil }
// XXX revise return value...
func (this *Cmd) Pipe(code string) (*Cmd, error) {
	return Run(code, this.Stdout) }
func (this *Cmd) Kill() *Cmd {
	if this.Cmd != nil {
		this.Process.Kill() }
	return this }

// helpers...
//
// XXX do we need these???
// XXX should these infirm to io.Writer, i.e. return nothing???
func (this *Cmd) Write(data []byte) *Cmd {
	this.Stdin.Write(data) 
	return this }
func (this *Cmd) WriteString(str string) (int, error) {
	// XXX how do we handle errors (.Stdin can be nil/closed/..)???
	return io.WriteString(this.Stdin, str) }



// vim:set ts=4 sw=4 nowrap :
