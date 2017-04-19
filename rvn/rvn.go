package rvn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

type Mount struct {
	Point, Source, Tag string
}

type Host struct {
	Name, OS string
	Mounts   []Mount
}

type Zwitch struct {
	Host
}

type Node struct {
	Host
}

type Endpoint struct {
	Name, Port string
}

type Link struct {
	Name      string
	Endpoints [2]Endpoint
}

type Topo struct {
	Name     string
	Nodes    []Node
	Switches []Zwitch
	Links    []Link
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

type Runtime struct {
	SubnetTable        [256]bool
	SubnetReverseTable map[string]int
}

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
	i := r.SubnetReverseTable[tag]
	r.SubnetTable[i] = false
	delete(r.SubnetReverseTable, tag)
	r.Save()
}
