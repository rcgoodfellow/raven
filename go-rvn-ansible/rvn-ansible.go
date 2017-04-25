package main

import (
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

	out, err := exec.Command(
		"ansible-playbook",
		"-i", ds.IP+",",
		yml,
		"--extra-vars", extra_vars,
		`--ssh-extra-args='-i/var/rvn/ssh/rvn'`,
		"--user=rvn",
	).CombinedOutput()

	if err != nil {
		log.Fatalf("ansible failed %s - %v", string(out), err)
	} else {
		log.Printf("%s", out)
	}

}

func usage() {
	fmt.Println("rvn-ansible <system> <node> <yml>")
}
