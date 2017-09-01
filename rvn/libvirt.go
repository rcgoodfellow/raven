package rvn

import (
	"encoding/json"
	"fmt"
	"github.com/libvirt/libvirt-go"
	xlibvirt "github.com/libvirt/libvirt-go-xml"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	//"time"
)

var conn *libvirt.Connect

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
		//log.Printf("connection nil - reconnecting")
		connect()
		//time.Sleep(1 * time.Millisecond)
	}

	for !isAlive() {
		//log.Printf("connection dead - reconnecting")
		connect()
		//time.Sleep(1 * time.Millisecond)
	}
}

/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * Public API Implementation
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

// Create creates a libvirt definition for the supplied topology.  It does not
// launch the system. For that functionality use the Launch function. If a
// topology with the same name as the topology provided as an argument exists,
// that topology will be overwritten by the system generated from the argument.

//TODO need to return an error if shizz goes sideways
func Create() {

	wd, err := WkDir()
	if err != nil {
		log.Printf("create: failed to get working dir")
		return
	}

	topo, err := loadTopo()
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

		for _, e := range link.Endpoints {
			d := doms[e.Name]
			h := topo.getHost(e.Name)
			if h == nil {
				log.Printf("unknown host in link %s", e.Name)
				continue
			}
			domConnect(topo.QualifyName(link.Name), topo.getHost(e.Name), d, link.Props)
		}

		nets[link.Name] = n
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

//TODO return error on sideways
func Destroy() {
	checkConnect()
	dbCheckConnection()

	wd, err := WkDir()
	if err != nil {
		log.Printf("newdom: failed to get working dir")
		return
	}

	topo, err := loadTopo()
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
		destroyNetwork(topo.QualifyName(x.Name), conn)
	}
	destroyNetwork(topo.QualifyName("test"), conn)
	LoadRuntime().FreeSubnet(topo.Name)
	UnexportNFS(topo.Name)
}

func Shutdown() []error {
	checkConnect()
	dbCheckConnection()

	topo, err := loadTopo()
	if err != nil {
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

// Launch brings up the system with the given name. This system must exist
// visa-vis the create function before calling Launch. The return value is
// a list of diagnostic strings that were provided by libvirt when launching
// the system. The existence of diagnostics does not necessarily indicate
// an error in launching. This function is asynchronous, when it returns the
// system is still launching. Use the Status function to check up on a the
// launch process.

//TODO name should probably be something more like 'deploy'
func Launch() []string {
	checkConnect()

	topo, err := loadTopo()
	if err != nil {
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

	topo, err := loadTopo()
	if err != nil {
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

/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * Helper functions
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

func newDom(h *Host, t *Topo) *xlibvirt.Domain {

	wd, err := WkDir()
	if err != nil {
		log.Printf("newdom: failed to get working dir")
		return nil
	}

	baseImage := "/var/rvn/img/" + h.Image + ".qcow2"
	instanceImage := wd + "/" + h.Name + ".qcow2"
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
		Memory: &xlibvirt.DomainMemory{Value: 4096, Unit: "MiB"},
		Devices: &xlibvirt.DomainDeviceList{
			Serials: []xlibvirt.DomainSerial{
				xlibvirt.DomainSerial{
					Type: "pty",
				},
			},
			Consoles: []xlibvirt.DomainConsole{
				xlibvirt.DomainConsole{
					Type:   "pty",
					Target: &xlibvirt.DomainConsoleTarget{Type: "serial"},
				},
			},
			Graphics: []xlibvirt.DomainGraphic{
				xlibvirt.DomainGraphic{
					Type:     "vnc",
					Port:     -1,
					AutoPort: "yes",
				},
			},
			Disks: []xlibvirt.DomainDisk{
				xlibvirt.DomainDisk{
					Type:   "file",
					Device: "disk",
					Driver: &xlibvirt.DomainDiskDriver{Name: "qemu", Type: "qcow2"},
					Source: &xlibvirt.DomainDiskSource{
						File: instanceImage,
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
			Type:   "network",
			Source: &xlibvirt.DomainInterfaceSource{Network: net},
			Model:  &xlibvirt.DomainInterfaceModel{Type: "virtio"},
			Boot:   boot,
		})
}

func loadTopo() (Topo, error) {

	/*
		topoDir := SysDir() + "/" + name
		path := topoDir + "/" + name + ".json"
	*/

	return LoadTopo()

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

func DomainStatus(topo, name string) (DomStatus, error) {
	checkConnect()
	return domainStatus(topo, name, topo+"_"+name, conn), nil
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
