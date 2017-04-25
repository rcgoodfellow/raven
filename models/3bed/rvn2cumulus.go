/*
 * rvn2cumulus
 * =====----~~
 *
 *   rvn2cumulus is a command line program that reads an expanded raven
 *   topology (e.g. the json not the js) and creates a set of Cumulus
 *	 interface configurations.
 *
 */
package main

import (
	"fmt"
	"github.com/rcgoodfellow/raven/rvn"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Ifx struct {
	For        string
	Name       string
	BridgeDefs []string
}

type SwitchConfig map[string][]Ifx

func usage() string {
	return "rvn2cumulus <topo.json>"
}

func main() {

	//no timestamp on logs
	log.SetFlags(0)

	//check command line args
	if len(os.Args) < 2 {
		log.Fatal(usage())
	}

	//read the topology file
	topo, err := rvn.LoadTopo(os.Args[1])
	if err != nil {
		log.Fatalf("failed to load topo %v", err)
	}

	//initailze empty set of configs
	configs := make(map[string][]Ifx)

	//iterate through  the links of the topology populating the configs
	//as we go
	for _, l := range topo.Links {

		//figure out which end is the switch
		var z, n rvn.Endpoint
		a := l.Endpoints[0]
		b := l.Endpoints[1]
		if strings.Contains(a.Name, "stem") || strings.Contains(a.Name, "leaf") {
			z = a
			n = b
		} else {
			z = b
			n = a
		}

		//build a switch interface config based on what type of node is hanging
		//off of it
		if n.Name == "boss" {
			configs[z.Name] = append(configs[z.Name], bossConfig(z.Port))
		} else if n.Name == "users" {
			configs[z.Name] = append(configs[z.Name], usersConfig(z.Port))
		} else if n.Name == "router" {
			configs[z.Name] = append(configs[z.Name], routerConfig(z.Port))
		} else {
			configs[z.Name] = append(configs[z.Name],
				nodeConfig(z.Name, z.Port, n.Name))
		}
	}

	//load the cumulus interface template file
	tp_path, err := filepath.Abs("./cumulus-interfaces-template")
	if err != nil {
		log.Fatalf("Failed to read template %v", err)
	}
	tp, err := template.ParseFiles(tp_path)
	if err != nil {
		log.Fatalf("failed to parse template %v", err)
	}

	//iterate through the configs we created for each swich using the template
	//to generate the associated cumulus linux configuration
	for zwitch, ifxs := range configs {

		outpath := fmt.Sprintf("%s-interfaces", zwitch)
		f, err := os.Create(outpath)
		if err != nil {
			log.Fatalf("failed to create output file %s - %v", outpath, err)
		}

		defer f.Close()
		err = tp.Execute(f, ifxs)
		if err != nil {
			log.Fatalf("failed to execute template %v", err)
		}

	}

}

//boss gets attached almost everything in trunk mode
func bossConfig(port string) Ifx {
	return Ifx{
		For:  "boss",
		Name: port,
		BridgeDefs: []string{
			"bridge-vids 2002 2003 2004 2006 2007",
			"bridge-allow-untagged no",
		}}
}

//users gets attached to a few things in trunk mode
func usersConfig(port string) Ifx {
	return Ifx{
		For:  "users",
		Name: port,
		BridgeDefs: []string{
			"bridge-vids 2002 2003 2005",
			"bridge-allow-untagged no",
		}}
}

//boss gets attached everything in trunk mode
func routerConfig(port string) Ifx {
	return Ifx{
		For:  "router",
		Name: port,
		BridgeDefs: []string{
			"bridge-vids 2002 2003 2004 2005 2006 2007",
			"bridge-allow-untagged no",
		}}
}

//all other classes of nodes are relegated to access on 2003
func nodeConfig(switchName, port, who string) Ifx {

	//control net defaults to 2003
	vlan := "2003"
	//experiment net defaults to 0
	if strings.Contains(switchName, "spine") ||
		strings.Contains(switchName, "leaf") {
		vlan = "0"
	}
	return Ifx{
		For:  who,
		Name: port,
		BridgeDefs: []string{
			"bridge-access " + vlan,
			"bridge-allow-untagged yes",
		}}
}
