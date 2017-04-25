package rvn

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	data := struct {
		Host Host
		NFS  string
	}{h, topo.MgmtIp}

	err = tp.Execute(f, data)
	if err != nil {
		log.Printf("failed to execute config template for %s - %v", h.Name, err)
	}
}

func Configure(topoName string) {
	topo, err := loadTopo(topoName)
	if err != nil {
		log.Println("configure: failed to load topo %s - %v", topoName, err)
		return
	}
	status := Status(topo.Name)
	node_status := status["nodes"].(map[string]DomStatus)
	switch_status := status["switches"].(map[string]DomStatus)

	var wg sync.WaitGroup
	doConfig := func(topo Topo, host Host, ds DomStatus) {
		yml := fmt.Sprintf("%s/%s/%s.yml", SysDir(), topo.Name, host.Name)
		log.Printf("running base config for %s:%s", topo.Name, host.Name)
		runConfig(yml, topo.Name, host, ds)

		user_yml := fmt.Sprintf("%s/%s.yml", topo.Dir, host.Name)
		if _, err := os.Stat(user_yml); err == nil {
			log.Printf("running user config for %s:%s", topo.Name, host.Name)
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

	extra_vars := "ansible_become_pass=rvn"
	if strings.ToLower(h.OS) == "freebsd" {
		extra_vars += " ansible_python_interpreter='/usr/local/bin/python'"
	}

	out, err := exec.Command(
		"ansible-playbook",
		"-i", s.IP+",",
		yml,
		"--extra-vars", extra_vars,
		`--ssh-extra-args='-i/var/rvn/ssh/rvn'`,
		"--user=rvn",
	).CombinedOutput()

	if err != nil {
		log.Printf("failed to run configuration for %s - %v", h.Name, err)
		log.Printf(string(out))
	}

}
