package rvn

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	path := fmt.Sprintf("/%s/%s/%s.yml", sysDir(), topo.Name, h.Name)
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
