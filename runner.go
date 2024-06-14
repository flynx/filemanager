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
	"bytes"
	//"sync"
	//"regexp"
	//"os"
)


var SHELL = "bash -c"


type Runner struct {
	*exec.Cmd

	Shell string

	State string
	Error error
	Done <-chan bool

	Stdout io.Reader
	Stderr io.Reader
}

func (this *Runner) makeEnv(){
	// XXX
}
func (this *Runner) Kill() *Runner {
	if this.Cmd != nil {
		this.Process.Kill() }
	return this }
func (this *Runner) Run(code string, stdin io.Reader) *Runner {
	shell := this.Shell
	if shell == "" {
		shell = SHELL }
	s := strings.Fields(shell)
	// setup the command...
	cmd := exec.Command(s[0], append(s[1:], code)...)
	this.Cmd = cmd
	//cmd.Env = this.makeEnv()

	cmd.Stdin = stdin

	//stdout := &bytes.Buffer{}
	stdout, _ := cmd.StdoutPipe()
	this.Stdout = stdout
	//cmd.Stdout = stdout

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	done := make(chan bool)
	this.Done = done
	// set state...
	// XXX do we need this???
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
			scanner := bufio.NewScanner(stderr)
			lines := []string{}
			for scanner.Scan() {
				lines = append(lines, scanner.Text()) }
			log.Println("    ERR:", strings.Join(lines, "\n"))
			//log.Println("    ENV:", env)
			this.Error = err
			done_state = false }
		done <- done_state }()

	return this }


/*
type Command struct {

	State string
	Done chan bool
	Kill chan bool
	Stdout *io.ReadCloser
	Stderr *io.ReadCloser
	Error error
}

func Run(code string, stdin io.Reader) Command {
	// build the command...
	s := this.Shell
	if s == "" {
		s = SHELL }
	shell := strings.Fields(s)
	cmd := exec.Command(shell[0], append(shell[1:], code)...)
	env := makeCallEnv(cmd)
	// io...
	cmd.Env = env
	// XXX can we make these optional???
	cmd.Stdin = stdin
	stdout, _ := cmd.StdoutPipe()
	// XXX for some reason we're not getting the error from the pipe...
	//stderr, _ := cmd.StderrPipe()
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	// output package...
	done := make(chan bool)
	kill := make(chan bool)
	res := Command {
		State: "pending",
		Done: done,
		Kill: kill,
		Stdout: &stdout, 
		//Stderr: &stderr,
	} 
	//SPINNER.Start()

	// handle killing the process when needed...
	watchdogDone := make(chan bool)
	go func(){
		//defer SPINNER.Stop()
		select {
			case <-kill:
				res.State = "killed"
				if err := cmd.Process.Kill() ; err != nil {
					log.Panic(err) } 
			case s := <-watchdogDone:
				if s == true {
					res.State = "done"
				} else {
					res.State = "failed" }
				return } }()

	// run...
	if err := cmd.Start(); err != nil {
		log.Panic(err) }

	// cleanup...
	go func(){
		done_state := true
		if err := cmd.Wait(); err != nil {
			log.Println("Error executing: \""+ code +"\"", err) 
			scanner := bufio.NewScanner(stderr)
			lines := []string{}
			for scanner.Scan() {
				lines = append(lines, scanner.Text()) }
			log.Println("    ERR:", strings.Join(lines, "\n"))
			log.Println("    ENV:", env)
			res.Error = err
			done_state = false }
		watchdogDone <- done_state
		done <- done_state }()

	return res }
//*/

