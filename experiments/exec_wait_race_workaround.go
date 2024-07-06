
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)


var RunCount = 100
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

		<-done
		if err := c.Wait(); err != nil {
			fmt.Println("!!! WAIT:", err)
		}
	}
	fmt.Println("")
	if s > 0 {
		fmt.Printf("Skipped part of the output %v times of %v\n", s, RunCount)
	}
}
