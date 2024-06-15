/*
*	
*
*/

package main

import (
	//"fmt"
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

type Cmd struct {
	*exec.Cmd

	Shell string
	Code string

	State string
	Error error
	Done <-chan bool

	Stdout io.Reader
	Stderr io.Reader

	LineHandler LineHandler
}

// XXX add ability to auto restart without losing context...
func Run(code string, stdin io.Reader) *Cmd {
	cmd := Cmd{
		Code: code,
	}
	cmd.Run(code, stdin)
	return &cmd }

type LineHandler func(string)
func (this *Cmd) HandleLine(handler LineHandler) *Cmd {
	this.LineHandler = handler
	go func(){
		scanner := bufio.NewScanner(this.Stdout)
		for scanner.Scan() {
			handler(scanner.Text()) } }() 
	return this }
// XXX
func (this *Cmd) makeEnv(){
	// XXX
}
// XXX make this restartable...
func (this *Cmd) Run(code string, stdin io.Reader) *Cmd {
	shell := this.Shell
	if shell == "" {
		shell = SHELL }
	s := strings.Fields(shell)
	// setup the command...
	cmd := exec.Command(s[0], append(s[1:], code)...)
	this.Cmd = cmd
	//cmd.Env = this.makeEnv()

	if stdin != nil {
		cmd.Stdin = stdin }
	var err error
	this.Stdout, err = cmd.StdoutPipe()
	if err != nil {
		log.Panic(err) }
	this.Stderr, err = cmd.StderrPipe()
	if err != nil {
		log.Panic(err) }

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
		log.Panic(err) }

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
		done <- done_state }()

	return this }
func (this *Cmd) Kill() *Cmd {
	if this.Cmd != nil {
		this.Process.Kill() }
	return this }


