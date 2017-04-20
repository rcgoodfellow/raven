package rvn

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"text/template"
)

func genConfig(h Host, topo Topo) {
	tp_path, err := filepath.Abs("../rvn/config.yml")
	if err != nil {
		log.Printf("failed to create absolute path for config.yml - %v", err)
		return
	}
	tp, err := template.ParseFiles(tp_path)
	if err != nil {
		log.Printf("failed to read config.yml - %v", err)
		return
	}

	path := fmt.Sprintf("/%s/%s/%s.yml", SysDir(), topo.Name, h.Name)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("failed to create path %s - %v", path, err)
		return
	}
	defer f.Close()

	err = tp.Execute(f, h)
	if err != nil {
		log.Printf("failed to execute config template for %s - %v", h.Name, err)
	}
}

func Configure(topoName string) {
	topo := loadTopo(topoName)
	status := Status(topo.Name)
	node_status := status["nodes"].(map[string]DomStatus)
	switch_status := status["switches"].(map[string]DomStatus)

	var wg sync.WaitGroup
	doConfig := func(topo Topo, host Host, ds DomStatus) {
		yml := fmt.Sprintf("%s/%s/%s.yml", SysDir(), topo.Name, host.Name)
		runConfig(yml, topo.Name, host, ds)

		user_yml := fmt.Sprintf("%s/%s.yml", topo.Dir, host.Name)
		if _, err := os.Stat(user_yml); err == nil {
			runConfig(user_yml, topo.Name, host, ds)
		}
		wg.Done()
	}

	for _, x := range topo.Nodes {
		wg.Add(1)
		go doConfig(topo, x.Host, node_status[x.Name])
	}
	for _, x := range topo.Switches {
		wg.Add(1)
		go doConfig(topo, x.Host, switch_status[x.Name])
	}

	wg.Wait()

	log.Println("configuration of all nodes complete")
}

func runConfig(yml, topo string, h Host, s DomStatus) {
	log.Printf("configuring %s:%s", topo, h.Name)
	out, err := exec.Command(
		"ansible-playbook",
		"-i", s.IP+",",
		yml,
		"--extra-vars", "ansible_become_pass=rvn",
		`--ssh-extra-args='-i/var/rvn/ssh/rvn'`,
		"--user=rvn",
	).Output()

	if err != nil {
		log.Printf("failed to run configuration for %s - %v", h.Name, err)
		log.Printf(string(out))
	}

}
