
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var RunCount = 1000

func main() {
	files, _ := os.ReadDir(".")
	c := len(files) + 2
	s := 0
	report := func(n int) {
		if n != c {
			fmt.Printf("\nListed: %v of %v (no errors)\n", n, c)
			s++
		} else {
			fmt.Print(".")
		}
	}
	for i := 0; i < RunCount; i++ {
		n := 0
		done := make(chan bool)
		c := exec.Command("ls", "-a")
		//c := exec.Command(strings.Fields("ls -a"))
		out, _ := c.StdoutPipe()
		go func() {
			scanner := bufio.NewScanner(out)
			for scanner.Scan() {
				scanner.Text()
				n++
			}
			// handle output...
			report(n)
			close(done)
		}()
		// start...
		if err := c.Start(); err != nil {
			fmt.Println("!!! START:", err)
		}
		// XXX this breaks the run script some of the time...
		// NOTE: there is an unpatched race in .Wait() so it mist either 
		//		be called when the command is done with it's output or
		//		the cleanup should be done manually...
		//		...doing this would force one of two architectural 
		//		approaches:
		//			- existing Go approach
		//				this would involve hooking into .Std***Pipe(..)
		//				and all the related mechanics either by 
		//				overloading or by re-writing...
		//				...this is very error prone...
		//			- callback
		//				...this can completely abstract the broblem away
		//				but will make things a bit less flexible...
		if err := c.Wait(); err != nil {
			fmt.Println("!!! WAIT:", err)
		}
		<-done
	}
	fmt.Println("")
	if s > 0 {
		fmt.Printf("Skipped part of the output %v times of %v\n", s, RunCount)
	}
}
