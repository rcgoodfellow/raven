package rvn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

// Types ======================================================================

type Mount struct {
	Point  string `json:"point"`
	Source string `json:"source"`
}

type UnitValue struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

type CPU struct {
	Sockets int    `json:"sockets"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
	Model   string `json:"arch"`
}

type Memory struct {
	Capacity UnitValue `json:"capacity"`
}

type Host struct {
	Name      string  `json:"name"`
	Image     string  `json:"image"`
	OS        string  `json:"os"`
	NoTestNet bool    `json:"no-testnet"`
	Mounts    []Mount `json:"mounts"`
	CPU       *CPU    `json:"cpu,omitempty"`
	Memory    *Memory `json:"memory,omitempty"`
}

type Zwitch struct {
	Host
}

type Node struct {
	Host
}

type Endpoint struct {
	Name string `json:"name"`
	Port string `json:"port"`
}

type Link struct {
	Name      string                 `json:"name"`
	Endpoints [2]Endpoint            `json:"endpoints"`
	Props     map[string]interface{} `json:"props"`
}

type Topo struct {
	Name     string   `json:"name"`
	Nodes    []Node   `json:"nodes"`
	Switches []Zwitch `json:"switches"`
	Links    []Link   `json:"links"`
	Dir      string   `json:"dir"`
	MgmtIp   string   `json:"mgmtip"`
}

type Runtime struct {
	SubnetTable        [256]bool
	SubnetReverseTable map[string]int
}

type RebootRequest struct {
	Topo  string   `json:"topo"`
	Nodes []string `json:"nodes"`
}

// Default Values =============================================================

var defaults = struct {
	Memory *Memory
	CPU    *CPU
}{
	Memory: &Memory{
		Capacity: UnitValue{
			Value: 4,
			Unit:  "GB",
		},
	},
	CPU: &CPU{
		Sockets: 1,
		Cores:   1,
		Threads: 1,
		Model:   "kvm64",
	},
}

func fillInMissing(h *Host) {
	if h.Memory == nil {
		h.Memory = defaults.Memory
	}
	if h.CPU == nil {
		h.CPU = defaults.CPU
	} else {
		//if these values are omitted by the user they are zero, but that is not
		//an appropriate default value e.g., there is no computer with 0 sockets
		if h.CPU.Sockets == 0 {
			h.CPU.Sockets = 1
		}
		if h.CPU.Cores == 0 {
			h.CPU.Cores = 1
		}
		if h.CPU.Threads == 0 {
			h.CPU.Threads = 1
		}
		if h.CPU.Model == "" {
			h.CPU.Model = "kvm64"
		}
	}

}

// Methods ====================================================================

// Topo -----------------------------------------------------------------------

func (t *Topo) getHost(name string) *Host {
	for i, x := range t.Nodes {
		if x.Name == name {
			return &t.Nodes[i].Host
		}
	}
	for i, x := range t.Switches {
		if x.Name == name {
			return &t.Switches[i].Host
		}
	}
	return nil
}

func (t Topo) QualifyName(n string) string {
	return t.Name + "_" + n
}

func (t Topo) String() string {
	s := t.Name + "\n"
	s += "nodes" + "\n"
	for _, v := range t.Nodes {
		s += fmt.Sprintf("  %+v\n", v)
	}
	s += "switches" + "\n"
	for _, v := range t.Switches {
		s += fmt.Sprintf("  %+v\n", v)
	}
	s += "links" + "\n"
	for _, v := range t.Links {
		s += fmt.Sprintf("  %+v\n", v)
	}
	return s
}

// Runtime --------------------------------------------------------------------

func (r *Runtime) Save() {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Printf("unable to marshal runtime state - %v", err)
		return
	}

	err = ioutil.WriteFile("/var/rvn/run", []byte(data), 0644)
	if err != nil {
		log.Printf("error saving runtime state - %v", err)
	}
}

func LoadRuntime() *Runtime {
	data, err := ioutil.ReadFile("/var/rvn/run")
	if err != nil {
		log.Fatalf("error reading rvn runtime file - %v", err)
	}

	rt := &Runtime{}
	err = json.Unmarshal(data, rt)
	if err != nil {
		log.Fatalf("error decoding runtime config - %v", err)
	}
	if rt.SubnetReverseTable == nil {
		rt.SubnetReverseTable = make(map[string]int)
	}
	return rt
}

func (r *Runtime) AllocateSubnet(tag string) int {
	i, ok := r.SubnetReverseTable[tag]
	if ok {
		return i
	}
	for i, b := range r.SubnetTable {
		if !b {
			r.SubnetTable[i] = true
			r.SubnetReverseTable[tag] = i
			r.Save()
			return i
		}
	}
	return -1
}

func (r *Runtime) FreeSubnet(tag string) {
	i, ok := r.SubnetReverseTable[tag]
	if ok {
		r.SubnetTable[i] = false
		delete(r.SubnetReverseTable, tag)
		r.Save()
	}
}

// Functions ==================================================================

func RunModel() error {

	// execute the javascript model
	out, err := exec.Command(
		"nodejs",
		"/usr/local/lib/rvn/run_model.js",
		"model.js",
	).CombinedOutput()

	if err != nil {
		log.Printf("error running model")
		log.Printf(string(out))
		return err
	}

	// save the result of the model execution in the working directory
	topo, err := ReadTopo(out)
	if err != nil {
		log.Printf("error reading topo %v", err)
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("cannot determine working directory %v", err)
		return err
	}
	topo.Dir = wd

	buf, err := json.MarshalIndent(topo, "", "  ")
	if err != nil {
		log.Printf("error marshalling topo %v", err)
		return err
	}

	err = ioutil.WriteFile(".rvn/topo.json", buf, 0644)
	if err != nil {
		log.Printf("error writing topo %v", err)
	}

	return nil

}

func LoadTopo() (Topo, error) {

	wd, err := WkDir()
	if err != nil {
		log.Printf("loadtopo: could not determine working directory")
		return Topo{}, err
	}

	path := wd + "/topo.json"
	return LoadTopoFile(path)
}

func LoadTopoFile(path string) (Topo, error) {

	f, err := ioutil.ReadFile(path)
	if err != nil {
		return Topo{}, err
	}
	topo, err := ReadTopo(f)
	if err != nil {
		return Topo{}, err
	}
	return topo, nil

}

func WkDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("wkdir: could not determine working directory %v", err)
		return "", err
	}
	return wd + "/.rvn", nil
}

func SrcDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("srcdir: could not determine working directory %v", err)
		return "", err
	}
	return wd, nil
}

func ReadTopo(src []byte) (Topo, error) {
	var topo Topo
	err := json.Unmarshal(src, &topo)
	if err != nil {
		return topo, err
	}

	// apply defaults to any values not supplied by user
	for i := 0; i < len(topo.Nodes); i++ {
		fillInMissing(&topo.Nodes[i].Host)
	}

	for i := 0; i < len(topo.Switches); i++ {
		fillInMissing(&topo.Switches[i].Host)
	}

	return topo, nil
}
