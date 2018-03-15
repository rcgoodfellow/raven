package rvn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
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

type Host struct {
	Name      string  `json:"name"`
	Image     string  `json:"image"`
	OS        string  `json:"os"`
	NoTestNet bool    `json:"no-testnet"`
	Mounts    []Mount `json:"mounts"`
	CPU       *CPU    `json:"cpu,omitempty"`
	Memory    *Memory `json:"memory,omitempty"`

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

	CheckRvnImages(topo)

	return topo, nil
}

func CheckRvnImages(topo Topo) {
	var images []string = make([]string, 0)
	// FIXME: NO EXTENTIONS!!
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
		parsedURL, _ := url.Parse(images[i])

		// NOTE: local images need to be prefixed with absolute path or ./ (accesible by root)
		// NOTE: image name cannot contain qcow2! qcow2 is added by rvn in libvirt.go
		remoteHost := parsedURL.Host

		// if there is no host to contact remotely, look locally, see note above on naming
		if remoteHost == "" {
			// check if this is local image path, or deterlab path
			splitPath := strings.Split(images[i], "/")
			if len(splitPath) > 1 {
				filePath := "/var/rvn/img/user/" + splitPath[len(splitPath)-1]
				_, err := os.Stat(filePath)
				// if there is an err, file does not exist, lets also check that the file
				// exists at the pathway, if it does, copy file to /var/rvn/img
				if err != nil {
					_, err = os.Stat(images[i])
					if err != nil {
						log.Fatalln("Copy failed - make sure if relative paths accessable by root (~)")
						log.Fatalln(err)
					} else {
						// no error, so image does exist, so lets copy from path to /var/rvn/img
						log.Println("Attempting copy from: " + images[i] + " to: " + filePath)
						err = CopyLocalFile(images[i], filePath)
						if err != nil {
							log.Fatalln(err)
						}
					}
				}
				// is only given by a name, so we can therefore assume it is a deterlab.net image
			} else {
				filePath := "/var/rvn/img/" + images[i]
				_, err := os.Stat(filePath)
				// if there is an err, file does not exist, download it
				if err != nil {
					// FIXME: shhh dont tell ryan about the extentions, need to rehost rvn qcows
					remotePath := "https://mirror.deterlab.net/rvn/img/" + images[i] + ".qcow2"
					log.Println("Attempting copy from: " + remotePath + " to: " + filePath)
					var dl_err error = DownloadFile(filePath, remotePath)
					// we tried to find the image on deterlab mirror, but could not, error
					if dl_err != nil {
						log.Fatalln(dl_err)
					}
				}
			}
			// if the host is remote, we should assume we are going off-world to get image
		} else {
			subPath, imageName, _ := ParseURL(parsedURL)
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
				dl_err := DownloadURL(parsedURL, filePath, imageName)
				if dl_err != nil {
					log.Fatalln(dl_err)
				}
			}
		}
	}
}

// https://gist.github.com/elazarl/5507969
func CopyLocalFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

// return a path, which we will create a directory tree with path[0]/path[1]/.../path[n]/image
func ParseURL(parsedURL *url.URL) (path string, image string, err error) {
	// Path is easier to use than RawPath
	remoteFullPath := parsedURL.Path
	splitPath := strings.Split(remoteFullPath, "/")
	// get the image name, dont let user specify qcow2
	// when rvn goes beyond qcow2, need to use correct format
	image = splitPath[len(splitPath)-1]
	// get the scheme used
	// create necessary variables
	var userName string
	var hostName string
	// now to create a directory tree from the path, omit scheme and opaque
	if parsedURL.Opaque != "" {
		err = errors.New("Opaque URL not implemented")
		return path, image, err
	}
	if parsedURL.User != nil {
		userName = parsedURL.User.Username()
		path = userName + "/"
	}
	if parsedURL.Host != "" {
		hostName = parsedURL.Host
		path = path + hostName + "/"
	}
	// ftp://user@host:/path will become user/host/path.../
	pathMinusImage := strings.Join(splitPath[:len(splitPath)-1], "/")
	path += pathMinusImage + "/"
	return path, image, nil
}

func DownloadURL(parsedURL *url.URL, downloadPath string, imageName string) error {
	URIScheme := parsedURL.Scheme
	// if no scheme for downloading file is provided, default to https
	// TODO: enforce HTTPS -- do not allow http, redirect
	if URIScheme == "https" {
		DownloadFile(downloadPath+imageName, parsedURL.String())
	} else if URIScheme == "http" {
		err := errors.New("http is not supported, please use https!")
		return err
	} else if URIScheme == "" {
		DownloadFile(downloadPath+imageName, parsedURL.String())
	} else {
		err := errors.New(parsedURL.Scheme + " is not currently implemented!")
		return err
	}
	return nil
}

// https://golangcode.com/download-a-file-from-a-url/
func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
