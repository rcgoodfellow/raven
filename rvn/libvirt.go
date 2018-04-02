package rvn

// TODO
//   - better error diagnostics and propagation

import (
	"encoding/json"
	"fmt"
	librvnhelp "github.com/isi-lincoln/raven/rvnhelper"
	"github.com/libvirt/libvirt-go"
	xlibvirt "github.com/libvirt/libvirt-go-xml"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Types ======================================================================

// DomStatus encapsulates various information about a libvirt domain for
// purposes of serialization and presentation.
type DomStatus struct {
	Name        string
	State       string
	ConfigState string
	IP          string
	Macs        []string
	VNC         int
}

// Variables ==================================================================

var conn *libvirt.Connect

// Functions ==================================================================

// Create creates a libvirt definition for the supplied topology.  It does not
// launch the system. For that functionality use the Launch function. If a
// topology with the same name as the topology provided as an argument exists,
// that topology will be overwritten by the system generated from the argument.
func Create() {
	//TODO need to return an error if shizz goes sideways

	wd, err := WkDir()
	if err != nil {
		log.Printf("create: failed to get working dir")
		return
	}

	topo, err := LoadTopo()
	if err != nil {
		log.Printf("create failed to load topo")
		return
	}

	topoDir := wd
	os.MkdirAll(topoDir, 0755)

	doms := make(map[string]*xlibvirt.Domain)
	nets := make(map[string]*xlibvirt.Network)

	subnet := LoadRuntime().AllocateSubnet(topo.Name)
	topo.MgmtIp = fmt.Sprintf("172.22.%d.1", subnet)

	nets["test"] = &xlibvirt.Network{
		Name: topo.QualifyName("test"),
		IPs: []xlibvirt.NetworkIP{
			xlibvirt.NetworkIP{
				Address: topo.MgmtIp,
				Netmask: "255.255.255.0",
				DHCP: &xlibvirt.NetworkDHCP{
					Ranges: []xlibvirt.NetworkDHCPRange{
						xlibvirt.NetworkDHCPRange{
							Start: fmt.Sprintf("172.22.%d.2", subnet),
							End:   fmt.Sprintf("172.22.%d.254", subnet),
						},
					},
				},
			},
		},
		Domain: &xlibvirt.NetworkDomain{
			Name:      topo.Name + ".net",
			LocalOnly: "yes",
		},
		Forward: &xlibvirt.NetworkForward{
			Mode: "nat",
		},
	}

	for _, node := range topo.Nodes {
		d := newDom(&node.Host, &topo)
		GenConfig(node.Host, topo)
		doms[node.Name] = d
		if !node.NoTestNet {
			domConnect(topo.QualifyName("test"), &node.Host, d, nil)
		}
	}

	for _, zwitch := range topo.Switches {
		d := newDom(&zwitch.Host, &topo)
		GenConfig(zwitch.Host, topo)
		doms[zwitch.Name] = d
		if !zwitch.NoTestNet {
			domConnect(topo.QualifyName("test"), &zwitch.Host, d, nil)
		}
	}

	for _, link := range topo.Links {

		n := &xlibvirt.Network{
			Name:   topo.QualifyName(link.Name),
			Bridge: &xlibvirt.NetworkBridge{Delay: "0", STP: "off"},
		}

		nets[link.Name] = n

	}

	resolveLinks(&topo)

	for _, x := range topo.Nodes {
		for _, p := range x.ports {
			domConnect(
				topo.QualifyName(p.Link),
				&x.Host,
				doms[x.Name],
				topo.getLink(p.Link).Props)
		}
	}
	for _, x := range topo.Switches {
		for _, p := range x.ports {
			domConnect(
				topo.QualifyName(p.Link),
				&x.Host,
				doms[x.Name],
				topo.getLink(p.Link).Props)
		}
	}

	data, _ := json.MarshalIndent(topo, "", "  ")
	ioutil.WriteFile(topoDir+"/topo.json", []byte(data), 0644)

	checkConnect()

	for _, d := range doms {
		xml, err := d.Marshal()
		if err != nil {
			log.Printf("error marshalling domain %v", err)
		}
		ioutil.WriteFile(topoDir+"/dom_"+d.Name+".xml", []byte(xml), 0644)

		dd, err := conn.LookupDomainByName(d.Name)
		if err != nil {
			_, err := conn.DomainDefineXML(xml)
			if err != nil {
				log.Printf("error defining domain %v", err)
			}
		} else {
			dd.Destroy()
			dd.Undefine()
			conn.DomainDefineXML(xml)
			dd.Free()
		}
	}

	for _, n := range nets {
		xml, _ := n.Marshal()
		ioutil.WriteFile(topoDir+"/net_"+n.Name+".xml", []byte(xml), 0644)

		nn, err := conn.LookupNetworkByName(n.Name)
		if err != nil {
			conn.NetworkDefineXML(xml)
		} else {
			nn.Destroy()
			nn.Undefine()
			conn.NetworkDefineXML(xml)
			nn.Free()
		}
	}

	//create NFS exports
	ExportNFS(topo)

}

// Destroy completely wipes out a topology with the given name. If the system
// is running within libvirt it is torn down. The entire definition of the
// system is also removed from libvirt.
func Destroy() {
	//TODO return error on sideways
	checkConnect()
	dbCheckConnection()

	wd, err := WkDir()
	if err != nil {
		log.Printf("newdom: failed to get working dir")
		return
	}

	topo, err := LoadTopo()
	if err != nil {
		//log.Printf("destroy: failed to load topo - %v", err)
		//nothing to destroy
		return
	}
	topoDir := wd
	exec.Command("rm", "-rf", topoDir).Run()

	for _, x := range topo.Nodes {
		destroyDomain(topo.QualifyName(x.Name), conn)
		state_key := fmt.Sprintf("config_state:%s:%s", topo.Name, x.Name)
		db.Del(state_key)
	}
	for _, x := range topo.Switches {
		destroyDomain(topo.QualifyName(x.Name), conn)
		state_key := fmt.Sprintf("config_state:%s:%s", topo.Name, x.Name)
		db.Del(state_key)
	}

	for _, x := range topo.Links {
		cleanupLinkNetwork(topo.QualifyName(x.Name), conn)
		destroyNetwork(topo.QualifyName(x.Name), conn)
	}
	cleanupTestNetwork(topo.QualifyName("test"), conn)
	destroyNetwork(topo.QualifyName("test"), conn)
	LoadRuntime().FreeSubnet(topo.Name)
	UnexportNFS(topo.Name)
}

// Shutdown turns of a virtual machine gracefully
func Shutdown() []error {
	checkConnect()
	dbCheckConnection()

	topo, err := LoadTopo()
	if err != nil {
		if strings.Contains(err.Error(), "topo.json: no such file or directory") {
			log.Printf("Topology not built. Use `rvn build` first")
			return []error{}
		}
		return []error{fmt.Errorf("shutdown: failed to load topo")}
	}

	errs := []error{}
	for _, x := range topo.Nodes {
		err := shutdownDomain(topo.QualifyName(x.Name), conn)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, x := range topo.Switches {
		err := shutdownDomain(topo.QualifyName(x.Name), conn)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func WipeNode(topo Topo, name string) error {
	checkConnect()
	dbCheckConnection()
	h := topo.getHost(name)
	if h == nil {
		return fmt.Errorf("host %s does not exist", name)
	}
	destroyDomain(topo.QualifyName(name), conn)
	d := newDom(h, &topo)

	if !h.NoTestNet {
		domConnect(topo.QualifyName("test"), h, d, nil)
	}
	resolveLinks(&topo)
	for _, p := range h.ports {
		domConnect(
			topo.QualifyName(p.Link),
			h,
			d,
			topo.getLink(p.Link).Props)
	}

	xml, err := d.Marshal()
	if err != nil {
		return fmt.Errorf("error marshalling domain %v", err)
	}

	dom, err := conn.DomainDefineXML(xml)
	if err != nil {
		return fmt.Errorf("error defining domain %v", err)
	}
	return dom.Create()
}

// Launch brings up the system with the given name. This system must exist
// visa-vis the create function before calling Launch. The return value is
// a list of diagnostic strings that were provided by libvirt when launching
// the system. The existence of diagnostics does not necessarily indicate
// an error in launching. This function is asynchronous, when it returns the
// system is still launching. Use the Status function to check up on a the
// launch process.
func Launch() []string {
	//TODO name should probably be something more like 'deploy'
	checkConnect()

	topo, err := LoadTopo()
	if err != nil {
		if strings.Contains(err.Error(), "topo.json: no such file or directory") {
			log.Printf("Topology not built. Use `rvn build` first")
			return []string{}
		}
		err := fmt.Errorf("failed to load topo %v", err)
		return []string{fmt.Sprintf("%v", err)}
	}

	//collect all the doamins and networks first so we know everything we need
	//exists
	var errors []string
	var doms []*libvirt.Domain
	var nets []*libvirt.Network

	for _, x := range topo.Nodes {
		d, err := conn.LookupDomainByName(topo.QualifyName(x.Name))
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", x.Name, err))
		} else {
			doms = append(doms, d)
		}
	}
	for _, x := range topo.Switches {
		d, err := conn.LookupDomainByName(topo.QualifyName(x.Name))
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", x.Name, err))
		} else {
			doms = append(doms, d)
		}
	}

	for _, x := range topo.Links {
		n, err := conn.LookupNetworkByName(topo.QualifyName(x.Name))
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", x.Name, err))
		} else {
			nets = append(nets, n)
		}
	}

	//test network
	n, err := conn.LookupNetworkByName(topo.QualifyName("test"))
	allowRpcBind(n)
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s: %v", "test", err))
	} else {
		nets = append(nets, n)
	}

	for _, net := range nets {
		active, err := net.IsActive()
		if err == nil && !active {
			err := net.Create()
			name, _ := net.GetName()
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			}
			if name != topo.QualifyName("test") {
				setBridgeProperties(net)
			}
			net.Free()
		}
	}

	for _, dom := range doms {
		active, err := dom.IsActive()
		if err == nil && !active {
			err := dom.Create()
			if err != nil {
				name, _ := dom.GetName()
				errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			}
			dom.Free()
		}
	}

	return errors
}

// Domain info fetches the libvirt domain information about host within a
// raven topology
func DomainInfo(topo, name string) (*xlibvirt.Domain, error) {

	checkConnect()
	dom, err := conn.LookupDomainByName(topo + "_" + name)
	if err != nil {
		return nil, err
	}
	xmldoc, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	xdom := &xlibvirt.Domain{}
	err = xdom.Unmarshal(xmldoc)
	if err != nil {
		return nil, err
	}

	return xdom, nil

}

// The status function returns the runtime status of a topology, node by node
// and network by network.
func Status() map[string]interface{} {

	status := make(map[string]interface{})

	checkConnect()

	topo, err := LoadTopo()
	if err != nil {
		if strings.Contains(err.Error(), "topo.json: no such file or directory") {
			log.Printf("Topology not built. Use `rvn build` first")
			return nil
		}
		log.Printf("status: failed to load topo - %v", err)
		return nil
	}

	nodes := make(map[string]DomStatus)
	status["nodes"] = nodes

	switches := make(map[string]DomStatus)
	status["switches"] = switches

	links := make(map[string]string)
	status["links"] = links

	for _, x := range topo.Nodes {
		nodes[x.Name] =
			domainStatus(topo.Name, x.Name, topo.QualifyName(x.Name), conn)
	}
	for _, x := range topo.Switches {
		switches[x.Name] =
			domainStatus(topo.Name, x.Name, topo.QualifyName(x.Name), conn)
	}

	for _, x := range topo.Links {
		links[x.Name] = networkStatus(topo.QualifyName(x.Name), conn)
	}

	subnet, ok := LoadRuntime().SubnetReverseTable[topo.Name]
	if ok {
		status["mgmtip"] = fmt.Sprintf("172.22.%d.1", subnet)
	}

	return status
}

// Domain status returns the current state of a domain (host) within a libvirt
// topology.
func DomainStatus(topo, name string) (DomStatus, error) {
	checkConnect()
	return domainStatus(topo, name, topo+"_"+name, conn), nil
}

// Reboot attempts to gracefully reboot a raven host
func Reboot(rr RebootRequest) error {
	checkConnect()

	topo, err := LoadTopo()
	if err != nil {
		return err
	}

	for _, x := range rr.Nodes {

		d, err := conn.LookupDomainByName(fmt.Sprintf("%s_%s", rr.Topo, x))
		if err != nil {
			continue
		}

		//seabios is not responsive to a reboot request, have to pull the plug
		h := topo.getHost(x)
		if h != nil && h.OS == "netboot" {
			d.Reset(0)
		} else {
			d.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT)
		}

	}

	return nil

}

// Helper Functions ===========================================================

func connect() {
	var err error
	conn, err = libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.Printf("libvirt connect failure: %v", err)
	}
}

func isAlive() bool {
	result, err := conn.IsAlive()
	if err != nil {
		log.Printf("error assesing connection liveness - %v", err)
		return false
	}
	return result
}

func checkConnect() {
	for conn == nil {
		connect()
	}

	for !isAlive() {
		connect()
	}
}

func newDom(h *Host, t *Topo) *xlibvirt.Domain {

	wd, err := WkDir()
	if err != nil {
		log.Printf("newdom: failed to get working dir")
		return nil
	}

	// location of baseImage depends on how image was imported
	baseImage := "/var/rvn/img/"
	// the instance name/location will depend on how it is parsed
	// if name is empty, then load netboot from /rvn/img
	if h.Image == "" {
		baseImage += "netboot"
		// if name points to a local path or to url
	} else if len(strings.Split(h.Image, "/")) > 1 {
		baseImage += "user/"
		parsedURL, _ := url.Parse(h.Image)
		remoteHost := parsedURL.Host
		// if remoteHost is empty, its a local image
		if remoteHost == "" {
			path := strings.Split(h.Image, "/")
			baseImage += path[len(path)-1]
		} else {
			subPath, imageName, _ := librvnhelp.ParseURL(parsedURL)
			baseImage += subPath + imageName
		}
		// this only leaves names, which default to deterlab and /rvn/img location
	} else {
		baseImage += h.Image
	}

	instanceImage := wd + "/" + h.Name
	exec.Command("rm", "-f", instanceImage).Run()

	out, err := exec.Command(
		"qemu-img",
		"create",
		"-f",
		"qcow2",
		"-o", "backing_file="+baseImage,
		instanceImage).CombinedOutput()

	if err != nil {
		log.Printf("error creating image file for %s", h.Name)
		log.Printf("%v", err)
		log.Printf("%s", out)
	}

	d := &xlibvirt.Domain{
		Type: "kvm",
		Name: t.QualifyName(h.Name),
		Features: &xlibvirt.DomainFeatureList{
			ACPI: &xlibvirt.DomainFeature{},
			APIC: &xlibvirt.DomainFeatureAPIC{},
		},
		OS: &xlibvirt.DomainOS{
			Type: &xlibvirt.DomainOSType{Type: "hvm"},
		},
		CPU: &xlibvirt.DomainCPU{
			/*
			   TODO: prefer a bit more discrimination than pure passthrough ....

			   Match: "minimum",
			   Model: &xlibvirt.DomainCPUModel{
			       Value: h.CPU.Model,
			   },
			*/
			Mode: "host-passthrough",
			Topology: &xlibvirt.DomainCPUTopology{
				Sockets: h.CPU.Sockets,
				Cores:   h.CPU.Cores,
				Threads: h.CPU.Threads,
			},
		},
		VCPU: &xlibvirt.DomainVCPU{
			Value: h.CPU.Sockets * h.CPU.Cores * h.CPU.Threads,
		},
		Memory: &xlibvirt.DomainMemory{
			Value: uint(h.Memory.Capacity.Value),
			Unit:  h.Memory.Capacity.Unit,
		},
		Devices: &xlibvirt.DomainDeviceList{
			Serials: []xlibvirt.DomainSerial{
				xlibvirt.DomainSerial{
					Source: &xlibvirt.DomainChardevSource{
						Pty: &xlibvirt.DomainChardevSourcePty{},
					},
				},
			},
			Consoles: []xlibvirt.DomainConsole{
				xlibvirt.DomainConsole{
					Source: &xlibvirt.DomainChardevSource{
						Pty: &xlibvirt.DomainChardevSourcePty{},
					},
					Target: &xlibvirt.DomainConsoleTarget{Type: "serial"},
				},
			},
			Graphics: []xlibvirt.DomainGraphic{
				xlibvirt.DomainGraphic{
					VNC: &xlibvirt.DomainGraphicVNC{
						Port:     -1,
						AutoPort: "yes",
					},
				},
			},
			Disks: []xlibvirt.DomainDisk{
				xlibvirt.DomainDisk{
					Device: "disk",
					Driver: &xlibvirt.DomainDiskDriver{Name: "qemu", Type: "qcow2"},
					Source: &xlibvirt.DomainDiskSource{
						File: &xlibvirt.DomainDiskSourceFile{
							File: instanceImage,
						},
					},
					Target: &xlibvirt.DomainDiskTarget{Dev: "vda", Bus: "virtio"},
				},
			},
		},
	}

	return d
}

func domConnect(
	net string, h *Host, dom *xlibvirt.Domain, props map[string]interface{}) {

	var boot *xlibvirt.DomainDeviceBoot = nil
	if strings.ToLower(h.OS) == "netboot" {
		if props != nil {
			boot_order, ok := props["boot"]
			if ok {
				boot_order_num, ok := boot_order.(float64)
				if ok {
					boot = &xlibvirt.DomainDeviceBoot{
						Order: uint(boot_order_num),
					}
				}
			}
		}
	}
	dom.Devices.Interfaces = append(dom.Devices.Interfaces,
		xlibvirt.DomainInterface{
			Source: &xlibvirt.DomainInterfaceSource{
				Network: &xlibvirt.DomainInterfaceSourceNetwork{
					Network: net,
				},
			},
			Model: &xlibvirt.DomainInterfaceModel{Type: "virtio"},
			Boot:  boot,
		})
}

func resolveLinks(t *Topo) {

	// 'plug in' the link to each node
	for _, l := range t.Links {
		for _, e := range l.Endpoints {
			h := t.getHost(e.Name)
			h.ports = append(h.ports, Port{l.Name, e.Port})
		}
	}

	// sort the links at each node by index
	for i := 0; i < len(t.Nodes); i++ {
		n := &t.Nodes[i]
		sort.Slice(n.ports, func(i, j int) bool {
			return n.ports[i].Index < n.ports[j].Index
		})
	}

}

func domainStatus(
	topo, name, qname string, conn *libvirt.Connect,
) DomStatus {

	var status DomStatus
	status.Name = name
	x, err := conn.LookupDomainByName(qname)
	if err != nil {
		status.State = "non-existant"
	} else {
		info, _ := x.GetInfo()
		switch info.State {
		case libvirt.DOMAIN_NOSTATE:
			status.State = "nostate"
		case libvirt.DOMAIN_RUNNING:
			status.State = "running"
			addrs, err := x.ListAllInterfaceAddresses(
				libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
			if err == nil {

				for _, a := range addrs {
					status.Macs = append(status.Macs, a.Hwaddr)
				}

				if len(addrs) > 0 {
					ifx := addrs[0]
					if len(ifx.Addrs) > 0 {
						status.IP = ifx.Addrs[0].Addr
					}
				}

			}
			status.ConfigState = configStatus(topo, name)
		case libvirt.DOMAIN_BLOCKED:
			status.State = "blocked"
		case libvirt.DOMAIN_PAUSED:
			status.State = "paused"
		case libvirt.DOMAIN_SHUTDOWN:
			status.State = "shutdown"
		case libvirt.DOMAIN_CRASHED:
			status.State = "crashed"
		case libvirt.DOMAIN_PMSUSPENDED:
			status.State = "suspended"
		case libvirt.DOMAIN_SHUTOFF:
			status.State = "off"
		}
		x.Free()
	}
	return status
}

func configStatus(topo, name string) string {
	dbCheckConnection()
	state_key := fmt.Sprintf("config_state:%s:%s", topo, name)
	val, err := db.Get(state_key).Result()
	if err == nil {
		return val
	} else {
		return ""
	}
}

func networkStatus(name string, conn *libvirt.Connect) string {
	x, err := conn.LookupNetworkByName(name)
	if err != nil {
		return "non-existant"
	} else {
		active, _ := x.IsActive()
		if active {
			return "up"
		} else {
			return "down"
		}
		x.Free()
	}
	return "?"
}

func destroyDomain(name string, conn *libvirt.Connect) {
	x, err := conn.LookupDomainByName(name)
	if err != nil {
		//ok nothing to destroy
	} else {
		x.Destroy()
		x.Undefine()
		x.Free()
	}
}

func shutdownDomain(name string, conn *libvirt.Connect) error {
	x, err := conn.LookupDomainByName(name)
	if err != nil {
		return fmt.Errorf("request to shutdown unknown vm %s", name)
	} else {
		x.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN)
		active, err := x.IsActive()
		if err == nil && active {
			log.Printf("shutting down %s", name)
			return x.Shutdown()
		}
		return err
	}
}

func cleanupLinkNetwork(name string, conn *libvirt.Connect) {
	x, err := conn.LookupNetworkByName(name)
	if err != nil {
		//ok nothing to clean up
	} else {
		cleanupBOOTP(x)
	}
}

func cleanupTestNetwork(name string, conn *libvirt.Connect) {
	x, err := conn.LookupNetworkByName(name)
	if err != nil {
		//ok nothing to clean up
	} else {
		cleanupRpcBind(x)
	}
}

func destroyNetwork(name string, conn *libvirt.Connect) {
	x, err := conn.LookupNetworkByName(name)
	if err != nil {
		//ok nothing to destroy
	} else {
		x.Destroy()
		x.Undefine()
		x.Free()
	}
}

func setBridgeProperties(net *libvirt.Network) {
	allowLLDP(net)
	allowBOOTP(net)
}

func allowLLDP(net *libvirt.Network) {
	name, _ := net.GetName()
	br, err := net.GetBridgeName()
	if err != nil {
		log.Printf("error getting bridge for %s - %v", name, err)
		return
	}

	err = ioutil.WriteFile(
		fmt.Sprintf("/sys/class/net/%s/bridge/group_fwd_mask", br),
		[]byte("16384"),
		0644,
	)

	if err != nil {
		log.Printf("unable to set group forwarding mask on bridge %s - %v",
			name,
			err,
		)
		return
	}
}

func allowBOOTP(net *libvirt.Network) {
	name, _ := net.GetName()
	br, err := net.GetBridgeName()
	if err != nil {
		log.Printf("error getting bridge for %s - %v", name, err)
		return
	}

	out, err := exec.Command("iptables", "-A", "FORWARD",
		"-i", br,
		"-d", "255.255.255.255",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		log.Printf("error allowing bootp through iptables %s - %v", out, err)
		return
	}

}

func cleanupBOOTP(net *libvirt.Network) {
	name, _ := net.GetName()
	br, err := net.GetBridgeName()
	if err != nil {
		log.Printf("error getting bridge for %s - %v", name, err)
		return
	}

	out, err := exec.Command("iptables", "-D", "FORWARD",
		"-i", br,
		"-d", "255.255.255.255",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		// don't bother reporting bad rule, that just means the rule does not exist
		// and there is nothing to do. This can happen when a system is built but not
		// run for example because the iptables rules only get created on launch
		if strings.Contains(string(out), "Bad rule") {
			return
		}
		log.Printf("error cleaning bootp iptables rules %s - %v", out, err)
		return
	}

}

func allowRpcBind(net *libvirt.Network) {
	name, _ := net.GetName()
	br, err := net.GetBridgeName()
	if err != nil {
		log.Printf("error getting bridge for %s - %v", name, err)
		return
	}

	out, err := exec.Command("iptables", "-I", "INPUT",
		"-i", br,
		"-p", "tcp",
		"--dport", "111",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		log.Printf("error allowing rpcbind tcp through iptables %s - %v", out, err)
		return
	}

	out, err = exec.Command("iptables", "-I", "INPUT",
		"-i", br,
		"-p", "udp",
		"--dport", "111",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		log.Printf("error allowing rpcbind udp through iptables %s - %v", out, err)
		return
	}

	out, err = exec.Command("iptables", "-I", "INPUT",
		"-i", br,
		"-p", "tcp",
		"--dport", "2049",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		log.Printf("error allowing nfs through iptables %s - %v", out, err)
		return
	}

}

func cleanupRpcBind(net *libvirt.Network) {
	name, _ := net.GetName()
	br, err := net.GetBridgeName()
	if err != nil {
		log.Printf("error getting bridge for %s - %v", name, err)
		return
	}

	out, err := exec.Command("iptables", "-D", "INPUT",
		"-i", br,
		"-p", "tcp",
		"--dport", "111",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		// don't bother reporting bad rule, that just means the rule does not exist
		// and there is nothing to do. This can happen when a system is built but not
		// run for example because the iptables rules only get created on launch
		if strings.Contains(string(out), "Bad rule") {
			return
		}
		log.Printf("error cleaning up iptables rpcbind tcp rule %s - %v", out, err)
		return
	}

	out, err = exec.Command("iptables", "-D", "INPUT",
		"-i", br,
		"-p", "udp",
		"--dport", "111",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		log.Printf("error cleaning up iptables rpcbind udp rule %s - %v", out, err)
		return
	}

	out, err = exec.Command("iptables", "-D", "INPUT",
		"-i", br,
		"-p", "tcp",
		"--dport", "2049",
		"-j", "ACCEPT").CombinedOutput()

	if err != nil {
		log.Printf("error cleaning up nfs iptables tcp rule%s - %v", out, err)
		return
	}

}
