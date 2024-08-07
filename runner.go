
package main

import (
	//"log"
	"fmt"
	//"os"
	"os/exec"
	"os/signal"
	"syscall"
	"sync"
	"io"
	"bufio"
	//"time"
	"strings"
	"bytes"
	"errors"
)


type LineHandler func(string) bool

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
// XXX need a way to kill this...
func Tee(reader io.Reader, writer io.Writer, handler LineHandler) (<-chan bool) {
	done := make(chan bool)
	buf := bytes.Buffer{}
	prebuf := bytes.Buffer{}
	var copying sync.Mutex
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		txt := scanner.Text()
		if handler != nil &&
				// stop if returning false...
				! handler(txt) {
			break }
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
	// cleanup...
	go func(){
		copying.Lock()
		// flush the pre-buffer if non-empty...
		if writer != nil && 
				prebuf.Len() > 0 {
			io.Copy(writer, &prebuf) } 
		close(done) }() 
	return done }

func TeeCloser(reader io.Reader, writer io.WriteCloser, handler LineHandler) (<-chan bool) {
	done := Tee(reader, writer, handler) 
	if writer != nil {
		go func(){
			<-done
			writer.Close() }() }
	return done }


// XXX use/write a generic Async(func, ...args)
func AsyncTee(reader io.Reader, writer io.Writer, handler LineHandler) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		Tee(reader, writer, handler)
		close(done) }()
	return done }
func AsyncTeeCloser(reader io.Reader, writer io.WriteCloser, handler LineHandler) (<-chan bool) {
	done := make(chan bool)
	go func(){ 
		TeeCloser(reader, writer, handler)
		close(done) }()
	return done }




var SHELL = "bash -c"
var PREFIX = "stdbuf -i0 -oL -eL"

type Cmd struct {
	*exec.Cmd

	Shell string
	Prefix string
	Code string

	Handler LineHandler

	Stdin io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser

	//Next PipedCmd

	Done <- chan bool
	__done chan bool

	// XXX do we actually need this???
	__resetting sync.Mutex
	__state_change sync.Mutex
}
func (this *Cmd) __reset(){
	this.__resetting.Lock()
	defer this.__resetting.Unlock()
	this.Cmd = nil 
	this.Stdin = nil 
	if this.__done != nil {
		close(this.__done) 
		this.__done = nil } }


func (this *Cmd) Run(stdin ...io.Reader) error {
	this.__state_change.Lock()
	defer this.__state_change.Unlock()

	if this.Cmd != nil {
		return errors.New(".Run(..): previous command not done.") }

	done := make(chan bool)
	// XXX these two are needed to have different types (ro vs rw)
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

	// kill children when killed...
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	//cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	// prevent children from zombifiying when killed...
	signal.Ignore(syscall.SIGCHLD)

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
		// we've been killed...
		if this.Cmd == nil {
			return }
		this.Cmd.Wait()
		this.__reset() }() 

	return this.Cmd.Start() }
func (this *Cmd) Wait() error {
	<-this.Done
	return nil }
func (this *Cmd) Kill() error {
	this.__state_change.Lock()
	defer this.__state_change.Unlock()
	defer this.__reset()
	if this.Cmd == nil {
		return errors.New(".Kill(..): no command running.") }
	// XXX not sure if we need this...
	ignorePanicClose := func(b io.Closer) error {
		defer func(){
			recover() }()
		if b == nil {
			return nil }
		return b.Close() }
	ignorePanicClose(this.Stdin)
	ignorePanicClose(this.Stdout)
	ignorePanicClose(this.Stderr)
	//log.Println("  -> kill:", this.Code)
	/*/
	this.Stdin.Close()
	this.Stdout.Close()
	this.Stderr.Close()
	//*/
	this.Process.Release()
	//return syscall.Kill(-this.Process.Pid, syscall.SIGKILL) }
	return this.Process.Kill() }
func (this *Cmd) Restart(stdin ...io.Reader) error {
	this.Kill()
	return this.Run(stdin...) }

// NOTE: this pipes .Stdout to the next command...
func (this *Cmd) PipeTo(code string, handler ...LineHandler) (*PipedCmd, error) {
	piped := PipedCmd{}
	piped.Code = code
	if len(handler) > 0 {
		piped.Handler = handler[0] }
	// run the piped command...
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
// XXX revise name -- should be both Go-ey and at the same time obvious, 
//		this does not fit the expectation from .Write(..) / io.Writer...
func (this *PipedCmd) Write(s string) (int, error) {
	if this.Stdin == nil {
		return 0, errors.New(fmt.Sprint(".Write(..): can not write to .Stdin:", this.Stdin)) }
	return io.WriteString(this.Stdin, s) }
func (this *PipedCmd) Writeln(s string) (int, error) {
	return this.Write(s +"\n") }

// XXX 
func (this *PipedCmd) Close() {
	// NOTE: after all the input is processed the unclosed .Stdin is the
	//		only thing that can keep .Wait() from releasing...
	this.Stdin.Close() }


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



// vim:set ts=4 sw=4 nowrap :
