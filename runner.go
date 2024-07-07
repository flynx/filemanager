
package main

import (
	"bufio"
	"fmt"
	//"os"
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
	__done chan bool
}
func (this *Cmd) __reset(){
	this.Cmd = nil 
	this.Stdin = nil 
	close(this.__done) }

func (this *Cmd) Run(stdin ...io.Reader) error {
	if this.Cmd != nil {
		return errors.New(".Run(..): previous command not done.") }

	done := make(chan bool)
	this.__done = done
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
		// XXX make this optional...
		scanner := bufio.NewScanner(this.Stdout)
		for scanner.Scan() {
			this.Handler(scanner.Text()) }
		this.Cmd.Wait()
		this.__reset() }()

	return this.Cmd.Start() }
func (this *Cmd) Wait() error {
	<-this.Done
	return nil }
func (this *Cmd) Kill() error {
	defer func(){
		this.__reset() }()
	if this.Cmd == nil {
		return errors.New(".Kill(..): no command running.") }
	return this.Process.Kill() }
// XXX revise???
func (this *Cmd) Restart(stdin ...io.Reader) error {
	this.Kill()
	return this.Run(stdin...) }

// XXX TEST...
func (this *Cmd) PipeTo(code string, handler LineHandler) (*PipedCmd, error) {
	piped := PipedCmd{}
	piped.Code = code
	piped.Handler = handler
	if err := piped.Run(this.Stdout); err != nil {
		return &piped, err }
	return &piped, nil }


func Run(code string, handler ...LineHandler) (*Cmd, error) {
	this := Cmd{}
	this.Code = code
	if len(handler) > 0 {
		this.Handler = handler[0] }
	if err := this.Run(); err != nil {
		return &this, err }
	return &this, nil }




// XXX Q: should we separate Cmd and PipedCmd???
//			...should we also overload .Run(..) ???
type PipedCmd struct {
	Cmd
}
func (this *PipedCmd) Run(stdin io.Reader) error {
	return this.Cmd.Run(stdin) }
// XXX revise name -- should be both Go-ey and at the same time obvious, 
//		this does not fit the expectation from .Write(..) / io.Writer...
func (this *PipedCmd) Write(s string) (int, error) {
	if this.Stdin == nil {
		return 0, errors.New(fmt.Sprint(".Write(..): can not write to .Stdin:", this.Stdin)) }
	return io.WriteString(this.Stdin, s) }
func (this *PipedCmd) Writeln(s string) (int, error) {
	return this.Write(s +"\n") }


func Pipe(code string, handler ...LineHandler) (*PipedCmd, error) {
	this := PipedCmd{}
	this.Code = code
	if len(handler) > 0 {
		this.Handler = handler[0] }
	r, w := io.Pipe()
	this.Stdin = w
	if err := this.Run(r); err != nil {
		return &this, err }
	go func(){
		this.Wait()
		w.Close() }()
	return &this, nil }



