package rvn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	xir "github.com/ceftb/xir/lang/go"
	log "github.com/sirupsen/logrus"
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

type Port struct {
	Link  string
	Index int
}

type Disktype struct {
	Bus string `json:"bus"`
	Dev string `json:"dev"`
}

type Host struct {
	Name            string    `json:"name"`
	Arch            string    `json:"arch"`
	Platform        string    `json:"platform"`
	Machine         string    `json:"machine"`
	Kernel          string    `json:"kernel"`
	Cmdline         string    `json:"cmdline"`
	Initrd          string    `json:"initrd"`
	Image           string    `json:"image"`
	OS              string    `json:"os"`
	NoTestNet       bool      `json:"no-testnet"`
	Mounts          []Mount   `json:"mounts"`
	CPU             *CPU      `json:"cpu,omitempty"`
	Memory          *Memory   `json:"memory,omitempty"`
	DefaultNic      string    `json:"defaultnic"`
	DefaultDisktype *Disktype `json:"defaultdisktype"`

	// internal use only
	ports []Port `json:"-"`
}

type Zwitch struct {
	Host
}

type Node struct {
	Host
}

type Endpoint struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

type Link struct {
	Name      string                 `json:"name"`
	Endpoints [2]Endpoint            `json:"endpoints"`
	Props     map[string]interface{} `json:"props"`
}

type Options struct {
	Display string `json:"display"`
}

type Topo struct {
	Name     string   `json:"name"`
	Nodes    []Node   `json:"nodes"`
	Switches []Zwitch `json:"switches"`
	Links    []Link   `json:"links"`
	Dir      string   `json:"dir"`
	MgmtIp   string   `json:"mgmtip"`
	Options  Options  `json:"options"`
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
//
// The default values are organized by platform. Each platform provides a basic
// set of default configuration variables sufficent to start a vm under that
// platform with no user specified configuration. Every new platform must
// support this runnable by convention model

type Platform struct {
	Name     string
	Arch     string
	Machine  string
	CPU      *CPU
	Memory   *Memory
	Kernel   string
	Image    string
	Cmdline  string
	Initrd   string
	Nic      string
	Disktype *Disktype
}

var defaults = struct {
	X86_64  *Platform
	Arm     *Platform
	Android *Platform
}{
	X86_64: &Platform{
		Name:     "x86_64",
		Arch:     "x86_64",
		Machine:  "pc-i440fx-2.10",
		CPU:      &CPU{Sockets: 1, Cores: 1, Threads: 1, Model: "kvm64"},
		Memory:   &Memory{Capacity: UnitValue{Value: 4, Unit: "GB"}},
		Image:    "netboot",
		Nic:      "virtio",
		Disktype: &Disktype{Dev: "vd", Bus: "virtio"},
	},
	Arm: &Platform{
		Name:     "arm7",
		Arch:     "armv7l",
		Machine:  "vexpress-a9",
		CPU:      &CPU{Sockets: 1, Cores: 1, Threads: 1, Model: "cortex-a9"},
		Memory:   &Memory{Capacity: UnitValue{Value: 1, Unit: "GB"}},
		Kernel:   "u-boot:a9",
		Image:    "raspbian:a9", //TODO s/raspbian/alpine/g
		Nic:      "virtio",
		Disktype: &Disktype{Dev: "sd", Bus: "sd"},
	},
	Android: &Platform{
		Name:     "android",
		Arch:     "x86_64",
		Machine:  "auto",
		CPU:      &CPU{Sockets: 1, Cores: 1, Threads: 1, Model: "kvm64"},
		Memory:   &Memory{Capacity: UnitValue{Value: 2, Unit: "GB"}},
		Image:    "oreo",
		Nic:      "virtio",
		Disktype: &Disktype{Dev: "vd", Bus: "virtio"},
	},
}

func fillInMissing(h *Host) {
	if h.Platform == "" {
		h.Platform = "x86_64"
	}

	switch h.Platform {

	case "x86_64":
		h.Arch = defaults.X86_64.Arch
		if h.Machine == "" {
			h.Machine = defaults.X86_64.Machine
		}
		applyCPUDefaults(&h.CPU, defaults.X86_64.CPU)
		applyMemoryDefaults(&h.Memory, defaults.X86_64.Memory)
		applyDisktypeDefaults(&h.DefaultDisktype, defaults.X86_64.Disktype)
		if h.Image == "" {
			h.Image = defaults.X86_64.Image
		}
		if h.DefaultNic == "" {
			h.DefaultNic = defaults.X86_64.Nic
		}

	case "arm7":
		h.Arch = defaults.Arm.Arch
		if h.Machine == "" {
			h.Machine = defaults.Arm.Machine
		}
		applyCPUDefaults(&h.CPU, defaults.Arm.CPU)
		applyMemoryDefaults(&h.Memory, defaults.Arm.Memory)
		applyDisktypeDefaults(&h.DefaultDisktype, defaults.Arm.Disktype)
		if h.Image == "" {
			h.Image = defaults.Arm.Image
		}
		if h.Kernel == "" {
			h.Kernel = defaults.Arm.Kernel
		}
		if h.DefaultNic == "" {
			h.DefaultNic = defaults.Arm.Nic
		}

	case "android":
		h.Arch = defaults.Android.Arch
		if h.Machine == "" {
			h.Machine = defaults.Android.Machine
		}
		applyCPUDefaults(&h.CPU, defaults.Android.CPU)
		applyMemoryDefaults(&h.Memory, defaults.Android.Memory)
		applyDisktypeDefaults(&h.DefaultDisktype, defaults.Android.Disktype)
		if h.Image == "" {
			h.Image = defaults.Android.Image
		}
		if h.DefaultNic == "" {
			h.DefaultNic = defaults.Android.Nic
		}

	}

}

func applyCPUDefaults(to **CPU, from *CPU) {
	if *to == nil {
		*to = new(CPU)
	}
	if (*to).Model == "" {
		(*to).Model = from.Model
	}
	if (*to).Sockets == 0 {
		(*to).Sockets = from.Sockets
	}
	if (*to).Cores == 0 {
		(*to).Cores = from.Cores
	}
	if (*to).Threads == 0 {
		(*to).Threads = from.Threads
	}
}

func applyMemoryDefaults(to **Memory, from *Memory) {
	if *to == nil {
		*to = new(Memory)
	}
	if (*to).Capacity.Value == 0 || (*to).Capacity.Unit == "" {
		(*to).Capacity = from.Capacity
	}
}

func applyDisktypeDefaults(to **Disktype, from *Disktype) {
	if *to == nil {
		*to = new(Disktype)
	}
	if (*to).Dev == "" {
		(*to).Dev = from.Dev
	}
	if (*to).Bus == "" {
		(*to).Bus = from.Bus
	}
}

func findKernel(h *Host) string {

	var defaultKernel string
	switch h.Arch {
	case "armv7l":
		defaultKernel = fmt.Sprintf("/var/rvn/kernel/%s", defaults.Arm.Kernel)
	case "Android":
		defaultKernel = ""
	case "x86_64":
		defaultKernel = ""
	default:
		defaultKernel = ""
	}

	if h.Kernel == "" {
		return defaultKernel
	}

	return findResource("kernel", h.Kernel, defaultKernel)

}

func findInitrd(h *Host) string {

	var defaultInitrd string
	switch h.Arch {
	case "armv7l":
		defaultInitrd = ""
	case "Android":
		defaultInitrd = ""
	case "x86_64":
		defaultInitrd = ""
	default:
		defaultInitrd = ""
	}

	if h.Initrd == "" {
		return defaultInitrd
	}

	return findResource("initrd", h.Initrd, defaultInitrd)

}

func findResource(kind, target, defaultValue string) string {

	// first: try to find the referenced resource in the local directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf(
			"find-%s: error getting working directory - using default", kind)
		return defaultValue
	}
	_, err = os.Stat(fmt.Sprintf("%s/%s", wd, target))
	if err == nil {
		return fmt.Sprintf("%s/%s", wd, target)
	}

	// second: if we cant find the referenced resource locally, try to find the
	// it in the rvn installation directory
	_, err = os.Stat(fmt.Sprintf("/var/rvn/%s/%s", kind, target))
	if err == nil {
		return fmt.Sprintf("/var/rvn/%s/%s", kind, target)
	}

	log.Printf(
		"find-%s: '%s' not found - using default", kind, target)

	return defaultValue
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

func (t *Topo) getLink(name string) *Link {
	for i, x := range t.Links {
		if x.Name == name {
			return &t.Links[i]
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

func Rvn2Xir(t *Topo) *xir.Net {

	net := xir.NewNet()

	for _, x := range t.Nodes {
		net.Node().Set(xir.Props{"name": x.Name})
	}
	for _, x := range t.Switches {
		net.Node().Set(xir.Props{"name": x.Name})
	}
	for _, x := range t.Links {
		a := net.GetNodeByName(x.Endpoints[0].Name)
		if a == nil {
			log.Errorf("bad node name %s", x.Endpoints[0].Name)
			continue
		}
		b := net.GetNodeByName(x.Endpoints[1].Name)
		if b == nil {
			log.Errorf("bad node name %s", x.Endpoints[1].Name)
			continue
		}
		net.Link(a.Endpoint(), b.Endpoint())
	}

	return net

}
