package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ceftb/xir/tools/viz"
	"github.com/fatih/color"
	"github.com/rcgoodfellow/raven/rvn"
	librvnhelp "github.com/rcgoodfellow/raven/rvnhelper"
	"github.com/sparrc/go-ping"
)

func main() {
	log.SetFlags(0)

	checkDir()

	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "build":
		doBuild()
	case "deploy":
		doDeploy()
	case "configure":
		doConfigure(os.Args[2:])
	case "shutdown":
		doShutdown()
	case "destroy":
		doDestroy()
	case "status":
		doStatus()
	case "viz":
		doViz()

	case "ssh":
		if len(os.Args) < 3 {
			usage()
		}
		doSsh(os.Args[2])
	case "ip":
		if len(os.Args) < 3 {
			usage()
		}
		doIp(os.Args[2])
	case "vnc":
		if len(os.Args) < 3 {
			usage()
		}
		doVnc(os.Args[2])
	case "ansible":
		if len(os.Args) < 4 {
			usage()
		}
		doAnsible(os.Args[2], os.Args[3])
	case "reboot":
		if len(os.Args) < 3 {
			usage()
		}
		doReboot(os.Args[2:])
	case "pingwait":
		if len(os.Args) < 3 {
			usage()
		}
		doPingwait(os.Args[2:])
	case "wipe":
		if len(os.Args) < 3 {
			usage()
		}
		doWipe(os.Args[2:])

	default:
		usage()
	}
}

func doBuild() {
	checkDir()
	err := rvn.RunModel()
	if err != nil {
		log.Fatal(err)
	}

	//error would happen in RunModel if LoadTopo failed
	topo, _ := rvn.LoadTopo()
	CheckRvnImages(topo)

	rvn.Create()
}

func doStatus() {

	status := rvn.Status()
	if status == nil {
		return
	}
	nodes := status["nodes"].(map[string]rvn.DomStatus)
	switches := status["switches"].(map[string]rvn.DomStatus)

	log.Println(blue("nodes"))
	for _, n := range nodes {
		log.Println(domString(n))
	}
	log.Println(blue("switches"))
	for _, s := range switches {
		log.Println(domString(s))
	}

}

func doConfigure(args []string) {
	if len(args) == 0 {
		rvn.Configure(true)
	} else {
		topo, err := rvn.LoadTopo()
		if err != nil {
			log.Fatal(err)
		}

		rvn.ConfigureNodes(topo, args)
	}
}

func doDeploy() {

	errors := rvn.Launch()
	if len(errors) != 0 {
		for _, e := range errors {
			log.Println(e)
		}
		os.Exit(1)
	}

}

func doShutdown() {

	errs := rvn.Shutdown()
	if errs != nil {
		for _, e := range errs {
			log.Printf("%v", e)
		}
		os.Exit(1)
	}

}

func doDestroy() {

	rvn.Destroy()

}

func doSsh(node string) {

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	ds, err := rvn.DomainStatus(topo.Name, node)
	if err != nil {
		fmt.Printf("error getting node status %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"ssh -o StrictHostKeyChecking=no -i /var/rvn/ssh/rvn rvn@%s\n", ds.IP)

}

func doViz() {

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	x := rvn.Rvn2Xir(&topo)

	viz.NetSvg(topo.Name, x)

}

func doIp(node string) {

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	ds, err := rvn.DomainStatus(topo.Name, node)
	if err != nil {
		fmt.Printf("error getting node status %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", ds.IP)

}

func doVnc(node string) {

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	di, err := rvn.DomainInfo(topo.Name, node)
	if err != nil {
		fmt.Printf("error getting domain info %v\n", err)
		os.Exit(1)
	}

	for _, x := range di.Devices.Graphics {
		if x.VNC != nil {
			fmt.Printf("%d\n", x.VNC.Port)
			break
		}
	}

}

func doAnsible(node, yml string) {

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	sys := topo.Name

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

	ds, err := rvn.DomainStatus(sys, node)
	if err != nil {
		fmt.Printf("error getting node status %v\n", err)
		os.Exit(1)
	}

	extra_vars := "ansible_become_pass=rvn"
	if strings.ToLower(h.OS) == "freebsd" {
		extra_vars += " ansible_python_interpreter='/usr/local/bin/python2'"
	}

	cmd := exec.Command(
		"ansible-playbook",
		"-i", ds.IP+",",
		yml,
		"--extra-vars", extra_vars,
		`--ssh-extra-args='-i/var/rvn/ssh/rvn'`,
		"--user=rvn", "--private-key=/var/rvn/ssh/rvn",
	)
	cmd.Env = append(os.Environ(), "ANSIBLE_HOST_KEY_CHECKING=False")

	reader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("failed to get stdout pipe %v", err)
	}
	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			log.Printf("%s\n", scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatalf("failed to start ansible command %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("failed to wait for ansible command to finish %v", err)
	}

}

func doReboot(args []string) {

	if len(args) < 1 {
		usage()
	}

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	rr := rvn.RebootRequest{
		Topo:  topo.Name,
		Nodes: args[0:],
	}

	rvn.Reboot(rr)

}

func doPingwait(args []string) {

	ipmap := make(map[string]string)

	// first wait until everything we need to ping has an IP
	success := false
	for !success {

		success = true

		status := rvn.Status()
		if status == nil {
			log.Fatal("could not query libvirt status")
		}

		nodes := status["nodes"].(map[string]rvn.DomStatus)
		switches := status["switches"].(map[string]rvn.DomStatus)
		// merge the switches into nodes since they can be treated the same in this
		// context
		for k, v := range switches {
			nodes[k] = v
		}

		for _, x := range args {
			n, ok := nodes[x]
			if !ok {
				log.Fatalf("%s does not exist", x)
			}
			if n.IP == "" {
				success = false
				break
			} else {
				ipmap[x] = n.IP
			}
		}

	}

	// now try to ping everything
	success = false

	for !success {

		success = true
		for _, x := range args {
			success = success && doPing(ipmap[x])
		}

	}

}

func doPing(host string) bool {

	p, err := ping.NewPinger(host)
	if err != nil {
		log.Fatal(err)
	}
	p.Count = 2
	p.Timeout = time.Millisecond * 500
	p.Interval = time.Millisecond * 50
	pings := 0
	p.OnRecv = func(pkt *ping.Packet) {
		pings++
	}
	p.Run()

	return pings == 2

}

func doWipe(args []string) {

	topo, err := rvn.LoadTopo()
	if err != nil {
		log.Fatal(err)
	}

	for _, x := range args {
		err = rvn.WipeNode(topo, x)
		if err != nil {
			log.Printf("%v", err)
		}
	}

}

func checkDir() {
	err := os.MkdirAll(".rvn", 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func domString(ds rvn.DomStatus) string {
	state := ds.State
	if state == "running" {
		state = green(state)
	}
	return fmt.Sprintf(
		"  %s %s %s %s", ds.Name, state, yellow(ds.ConfigState), ds.IP)
}

func usage() {
	s := red("usage:\n")
	s += fmt.Sprintf("  %s [%s | %s | %s | %s | %s | %s | %s] \n", blue("rvn"),
		green("build"),
		green("deploy"),
		green("configure"),
		green("shutdown"),
		green("destroy"),
		green("status"),
		green("viz"),
	)
	s += fmt.Sprintf("  %s %s node\n", blue("rvn"), green("ssh"))
	s += fmt.Sprintf("  %s %s node\n", blue("rvn"), green("ip"))
	s += fmt.Sprintf("  %s %s node\n", blue("rvn"), green("vnc"))
	s += fmt.Sprintf("  %s %s node-1 ... node-n\n", blue("rvn"), green("configure"))
	s += fmt.Sprintf("  %s %s node-1 ... node-n\n", blue("rvn"), green("reboot"))
	s += fmt.Sprintf("  %s %s node-1 ... node-n\n", blue("rvn"), green("pingwait"))
	s += fmt.Sprintf("  %s %s node-1 ... node-n\n", blue("rvn"), green("wipe"))
	s += fmt.Sprintf("  %s %s node script.yml", blue("rvn"), green("ansible"))

	log.Fatal(s)
}

var blueb = color.New(color.FgBlue, color.Bold).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()
var cyanb = color.New(color.FgCyan, color.Bold).SprintFunc()
var greenb = color.New(color.FgGreen, color.Bold).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var redb = color.New(color.FgRed, color.Bold).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var bold = color.New(color.Bold).SprintFunc()

func CheckRvnImages(topo rvn.Topo) {
	var images []string = make([]string, 0)
	// NOTE: moving over to an extentionless naming scheme
	// for each Node, get the image required
	for i := 0; i < len(topo.Nodes); i++ {
		var currentImg string = topo.Nodes[i].Host.Image
		// no way to check existance
		var exists bool = false
		for j := 0; j < len(images); j++ {
			if currentImg == images[j] {
				exists = true
				break
			}
		}
		// we did not find image, add it to our image list
		if exists != true {
			images = append(images, currentImg)
		}
	}
	// for each unique image, check that it exists, if not, download it
	for i := 0; i < len(images); i++ {
		// parse the uri reference (with golang url parser)
		parsedURL := librvnhelp.ValidateURL(images[i])

		// NOTE: local images need to be prefixed with absolute path or ./
		remoteHost := parsedURL.Host

		// there is no remote host, so the image is local
		if remoteHost == "" {
			// very first thing, check if url is nil or empty, then we need to create
			// a netboot image if it does not exist
			if parsedURL.String() == "" {
				err := librvnhelp.CreateNetbootImage()
				if err != nil {
					log.Fatalln(err)
				}
				break
			}
			// check if this is local image path, or deterlab path
			splitPath := strings.Split(images[i], "/")
			// a local image must be prefixed with atleast one slash, / or ./, therefore
			// we can check the length of of the split on / to determin if deterlab or local image
			if len(splitPath) > 1 {
				// place user images in /var/rvn/img/user
				filePath := "/var/rvn/img/user/" + splitPath[len(splitPath)-1]
				_, err := os.Stat(filePath)
				// if file exists, copy file to /var/rvn/img
				if err != nil {
					_, err = os.Stat(images[i])
					if err != nil {
						log.Fatalln("Copy failed - make sure if relative paths accessable by root (~)")
						log.Fatalln(err)
					} else {
						// no error, so image does exist, so lets copy from path to /var/rvn/img
						log.Println("Downloading from: " + images[i] + " to: " + filePath)
						err = librvnhelp.CopyLocalFile(images[i], filePath)
						if err != nil {
							log.Fatalln(err)
						}
					}
				}
				// if len == 1, we are given a name, we assume the named image is hosted on deterlab
			} else {
				// official images go in /var/rvn/img
				filePath := "/var/rvn/img/" + images[i]
				_, err := os.Stat(filePath)
				// if there is an err, file does not exist, download it
				if err != nil {
					remotePath := "https://mirror.deterlab.net/rvn/img/" + images[i]
					log.Println("Attempting copy from: " + remotePath + " to: " + filePath)
					var dl_err error = librvnhelp.DownloadFile(filePath, remotePath)
					// we tried to find the image on deterlab mirror, but could not, error
					if dl_err != nil {
						log.Fatalln(dl_err)
					}
				}
			}
			// if the host is remote, we should assume we are going off-world to get image
		} else {
			subPath, imageName, _ := librvnhelp.ParseURL(parsedURL)
			filePath := "/var/rvn/img/user/" + subPath
			_, err := os.Stat(filePath + imageName)
			// path to image does not exist, we will need to download it
			if err != nil {
				// first we create all the subdirectorys in the file path
				cr_err := os.MkdirAll(filePath, 0755)
				if cr_err != nil {
					log.Fatalln(cr_err)
				}
				// now try to download the image to the correct lcoation
				log.Println("Attempting copy from: " + parsedURL.String() + " to: " + filePath + imageName)
				dl_err := librvnhelp.DownloadURL(parsedURL, filePath, imageName)
				if dl_err != nil {
					log.Fatalln(dl_err)
				}
			}
		}
	}
}
