
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
	"slices"
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
func (this *Cmd) Reset(){
	this.__resetting.Lock()
	defer this.__resetting.Unlock()
	this.Cmd = nil 
	this.Stdin = nil 
	if this.__done != nil {
		close(this.__done) 
		this.__done = nil } }

//
//	.Init([<stdin>[, <stdout>[, <stderr>]]][, <env>])
//		-> error
//
func (this *Cmd) Init(args ...any) error {
	// parse args...
	var env []string
	for i, e := range args {
		if v, ok := e.(Env); ok {
			env = []string{}
			for k, v := range v {
				env = append(env, k +"="+ v) }
			args = slices.Delete(args, i, i+1) 
			break }
		if v, ok := e.([]string); ok {
			env = v
			args = slices.Delete(args, i, i+1) 
			break } }
	var stdin io.Reader
	if len(args) >= 1 && 
			args[0] != nil {
		stdin = args[0].(io.Reader) }
	var stdout io.WriteCloser
	if len(args) >= 2 && 
			args[1] != nil {
		stdout = args[1].(io.WriteCloser) }
	var stderr io.WriteCloser
	if len(args) >= 3 && 
			args[2] != nil {
		stderr = args[2].(io.WriteCloser) }

	if this.Cmd != nil {
		return errors.New(".Run(..): previous command not done.") }

	done := make(chan bool)
	// NOTE: these two are needed to have different types (ro vs rw)
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

	// env...
	if env != nil {
		cmd.Env = env }
	// stdin...
	if stdin != nil {
		cmd.Stdin = stdin }
	// stdout...
	if stdout != nil {
		cmd.Stdout = stdout
		if this.Stdout == nil {
			this.Stdout = stdout.(io.ReadCloser) }
	} else if p, err := cmd.StdoutPipe(); err == nil {
		this.Stdout = p
	} else {
		return err }
	// stderr...
	if stderr != nil {
		cmd.Stderr = stderr
		if this.Stderr == nil {
			this.Stderr = stderr.(io.ReadCloser) }
	} else if p, err := cmd.StderrPipe(); err == nil {
		this.Stderr = p
	} else {
		return err }

	return nil }
func (this *Cmd) Run(args ...any) error {
	this.__state_change.Lock()
	defer this.__state_change.Unlock()

	// init...
	if err := this.Init(args...); err != nil {
		return err }

	// kill children when killed...
	this.Cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	//this.Cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	// prevent children from zombifiying when killed...
	signal.Ignore(syscall.SIGCHLD)

	// handle the output (tee stdout)...
	if this.Handler != nil {
		src := this.Stdout
		r, w := io.Pipe()
		this.Stdout = r
		go func(){
			TeeCloser(src, w, this.Handler)
			// we've been killed...
			if this.Cmd == nil {
				return }
			this.Cmd.Wait()
			this.Reset() }() 
	// no handler -- cleanup...
	} else {
		go func(){
			this.Cmd.Wait()
			// XXX this for some reason breaks interactive commands (this.Cmd = nil)...
			this.Reset() }() }

	return this.Cmd.Start() }

func (this *Cmd) Wait() error {
	<-this.Done
	return nil }
func (this *Cmd) Kill() error {
	this.__state_change.Lock()
	defer this.__state_change.Unlock()
	defer this.Reset()
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
func (this *Cmd) Restart(args ...any) error {
	this.Kill()
	return this.Run(args...) }

// NOTE: this pipes .Stdout to the next command...
func (this *Cmd) PipeTo(code string, handler ...LineHandler) (*PipedCmd, error) {
	piped := PipedCmd{}
	// XXX can't set this declaratively for some reason...
	piped.Code = code
	if len(handler) > 0 {
		piped.Handler = handler[0] }
	// run the piped command...
	if err := piped.Run(this.Stdout); err != nil {
		return &piped, err }
	return &piped, nil }


//
//	Run(<code>[, <io.Reader>][, <env>][, <LineHandler>])
//		-> *Cmd, error
//
// XXX simply pass rest onto .Run(..)
func Run(code string, rest ...any) (*Cmd, error) {
	var stdin io.Reader
	var env []string
	this := Cmd{}
	// XXX can't set this declaratively via Cmd{ Code: .. } for some reason...
	this.Code = code
	for _, r := range rest {
		switch r.(type) {
			// XXX does this work??? (see notes in Pipe(..) below)
			case LineHandler:
				this.Handler = r.(LineHandler)
			case io.Reader:
				stdin = r.(io.Reader) 
			case Env:
				for k, v := range r.(Env) {
					env = append(env, k +"="+ v) }
			case []string:
				env = r.([]string)
			// XXX handle error...
			//default:
			//	log.Println("Error")
		} }
	if err := this.Run(stdin, env); err != nil {
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


// XXX should this be the same as Run(..)
// XXX for some magical reason replacing LineHandler with any in args and 
//		casting to LineHandler in the test does not work...
//		...this prevents us from reusing the code from Run...
//		.....printing the argument shows func(string)bool which is the 
//		same as LineHandler... BUG???
// XXX simply pass rest onto .Run(..)
func Pipe(code string, rest ...any) (*PipedCmd, error) {
	this := PipedCmd{}
	// XXX can't set this declaratively via Cmd{ Code: .. } for some reason...
	this.Code = code
	// XXX should this deafult to process env???
	env := []string{}
	for _, r := range rest {
		switch r.(type) {
			// XXX why does this not work, while the next case does???
			//		...does this work in .Run(..)???
			//case LineHandler:
			//	this.Handler = r.(LineHandler)
			case (func(string) bool):
				this.Handler = r.(func(string) bool)
			case []string:
				env = r.([]string)
			// XXX handle errors...
			//default:
			//	log.Println("Error")
		} }
	r, w := io.Pipe()
	this.Stdin = w
	if err := this.Run(r, env); err != nil {
		return &this, err }
	go func(){
		this.Wait()
		w.Close() }()
	return &this, nil }



// vim:set ts=4 sw=4 nowrap :
