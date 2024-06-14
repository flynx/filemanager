
package main

import (
	"testing"
	//"bytes"
	"bufio"
	//"strconv"
	"fmt"
)


func TestBasics(t *testing.T){
	r := Runner{}
	cmd := "pwd; sleep 1; ls"

	fmt.Println("$", cmd)
	//r.Run(cmd, &bytes.Buffer{})
	r.Run(cmd, nil)

	scanner := bufio.NewScanner(r.Stdout)
	for scanner.Scan() {
		fmt.Println(">>", scanner.Text()) }

	go func(){
		<-r.Done
		fmt.Println("done.")
	}()

	<-r.Done
}

