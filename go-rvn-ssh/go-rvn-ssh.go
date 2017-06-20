package main

import (
	"fmt"
	"github.com/rcgoodfellow/raven/rvn"
	"os"
)

func main() {

	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}
	sys := os.Args[1]
	node := os.Args[2]

	ds, err := rvn.DomainStatus(sys, node)
	if err != nil {
		fmt.Printf("error getting node status %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("ssh -o StrictHostKeyChecking=no -i /var/rvn/ssh/rvn rvn@%s\n", ds.IP)

}

func usage() {
	fmt.Println("rvn-ssh <system> <node>")
}
