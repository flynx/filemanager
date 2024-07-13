
package main

import (
	"bufio"
	"fmt"
	//"os"
	"os/exec"
	"sync"
	"io"
	//"time"
	"strings"
	"bytes"
	"errors"
)


// Tee the reader output into a function and a writer...
//
//	reader -> tee -> handler(line)
//				\
//				 +-> writer
//
// NOTE: the handler is called sync, this if it blocks it will block the 
//		line write to writer
//		XXX should this be the case???
// NOTE: this buffers the writes and will not block on the writer
// XXX can we somehow avoid the blocking goroutine that copies the buf 
//		to writer???
//		...somehow initiate this by the writer's reader...
//		....or stop it on exit...
func Tee(reader io.Reader, writer io.Writer, handler func(string)) {
	buf := bytes.Buffer{}
	prebuf := bytes.Buffer{}
	var copying sync.Mutex
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		txt := scanner.Text()
		if handler != nil {
			handler(txt) }
		// nothing is in queue to pipe -- flush pre-buffer, write...
		if writer != nil && 
				copying.TryLock() {
			buf.ReadFrom(&prebuf)
			buf.WriteString(txt +"\n")
			// write to output...
			go func(){
				defer copying.Unlock() 
				io.Copy(writer, &buf) }() 
		// waiting to write -- pre-buffer...
		} else {
			io.WriteString(&prebuf, txt +"\n") } }
	copying.Lock()
	// flush the pre-buffer if non-empty...
	if writer != nil && 
			prebuf.Len() > 0 {
		io.Copy(writer, &prebuf) } }

func TeeCloser(reader io.Reader, writer io.WriteCloser, handler func(string)) {
	if writer != nil {
		defer writer.Close() }
	Tee(reader, writer, handler) }

// XXX make this a generic Async(func, ...args)
func AsyncTee(reader io.Reader, writer io.Writer, handler func(string)) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		Tee(reader, writer, handler)
		close(done) }()
	return done }

// XXX make this a generic Async(func, ...args)
func AsyncTeeCloser(reader io.Reader, writer io.WriteCloser, handler func(string)) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		TeeCloser(reader, writer, handler)
		close(done) }()
	return done }




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

	//Next PipedCmd

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
	// XXX there are instances where we can want stdout to be a buffer 
	//		instead of a pipe...
	//		...for example a pipe makes the case where we do not read/pipe
	//		stdout allot more complicated...
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

	// handle the output...
	src := this.Stdout
	r, w := io.Pipe()
	this.Stdout = r
	go func(){
		TeeCloser(src, w, this.Handler)
		//w.Close()
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
func (this *Cmd) PipeTo(code string, handler ...LineHandler) (*PipedCmd, error) {
	piped := PipedCmd{}
	piped.Code = code
	if len(handler) > 0 {
		piped.Handler = handler[0] }

	// transfer output to piped (passive)...
	//if this.Handler == nil {
		this.Handler = func(s string){
			piped.Writeln(s) }
		if err := piped.Run(this.Stdout); err != nil {
			return &piped, err }
		go func(){
			<-piped.Done
			this.Cmd.Wait()
			this.__reset() }()
	/*/
	// tee output to local handler and piped (active)...
	// XXX this will not work as by this time this can be already running 
	//		and using the old handler...
	} else {
		local_handler := this.Handler
		// XXX need to do this BEFORE this starts, or to save the output someplace...
		this.Handler = func(s string){
			local_handler(s)
			piped.Writeln(s) } 
		if err := piped.Run(); err != nil {
			return &piped, err } }
	//*/

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



