package main

//FIXME
//  When there is an ansible syntax error, we just report an exit code 4 and
//  do not print the ansible diagnostics.

import (
	"bufio"
	"fmt"
	"github.com/rcgoodfellow/raven/rvn"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {

	log.SetFlags(0)

	if len(os.Args) < 4 {
		usage()
		os.Exit(1)
	}
	sys := os.Args[1]
	node := os.Args[2]
	yml := os.Args[3]

	topo, err := rvn.LoadTopoByName(sys)
	if err != nil {
		log.Fatalf("failed to load topology - %v", err)
	}
	var h *rvn.Host = nil
	for _, x := range topo.Nodes {
		if x.Name == node {
			h = &x.Host
			break
		}
	}
	if h == nil {
		for _, x := range topo.Switches {
			if x.Name == node {
				h = &x.Host
				break
			}
		}
	}
	if h == nil {
		log.Fatal("%s not found in topology", node)
	}

	ds, err := rvn.DomainStatus(sys + "_" + node)
	if err != nil {
		fmt.Printf("error getting node status %v\n", err)
		os.Exit(1)
	}

	extra_vars := "ansible_become_pass=rvn"
	if strings.ToLower(h.OS) == "freebsd" {
		extra_vars += " ansible_python_interpreter='/usr/local/bin/python'"
	}

	cmd := exec.Command(
		"ansible-playbook",
		"-i", ds.IP+",",
		yml,
		"--extra-vars", extra_vars,
		`--ssh-extra-args='-i/var/rvn/ssh/rvn'`,
		"--user=rvn",
	)

	reader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to get stdout pipe %v", err)
	}
	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			log.Printf("%s\n", scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatalf("failed to start ansible command %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("failed to wait for ansible command to finish %v", err)
	}

}

func usage() {
	fmt.Println("rvn-ansible <system> <node> <yml>")
}
