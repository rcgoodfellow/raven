package rvn

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"io/ioutil"
	"log"
	"os/user"
)

var db *redis.Client

func dbConnect() {
	db = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if db == nil {
		log.Fatal("failed to connect to db")
	}
}

func dbAlive() bool {
	_, err := db.Ping().Result()
	if err != nil {
		log.Printf("ping db failed")
		return false
	}
	return true
}

func dbCheckConnection() {
	for db == nil {
		dbConnect()
	}

	for !dbAlive() {
		dbConnect()
	}
}

type Mount struct {
	Point  string `json:"point"`
	Source string `json:"source"`
}

type Host struct {
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	OS     string  `json:"os"`
	Mounts []Mount `json:"mounts"`
	Level  int     `json:"level"`
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

func LoadTopo(path string) (Topo, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return Topo{}, err
	}
	return ReadTopo(f), nil
}

func LoadTopoByName(system string) (Topo, error) {
	path := fmt.Sprintf("%s/%s/%s.json", SysDir(), system, system)
	return LoadTopo(path)
}

func SysDir() string {
	u, err := user.Current()
	if err != nil {
		log.Printf("error getting user: %v", err)
	}
	return u.HomeDir + "/.rvn/systems"
}

func ReadTopo(src []byte) Topo {
	var topo Topo
	json.Unmarshal(src, &topo)
	return topo
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
