/*
 * rvn2cumulus
 * =====----~~
 *
 *   controlmacs is a command line program that reads an expanded raven
 *   topology (e.g. the json not the js) and creates a control mac to
 *	 node-type mapping file.
 *
 */

package main

import (
	"encoding/json"
	"github.com/rcgoodfellow/raven/rvn"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	log.SetFlags(0)

	//read the topology file
	topo, err := rvn.LoadTopo(os.Args[1])
	if err != nil {
		log.Fatalf("failed to load topo %v", err)
	}

	imap := make(map[string]string)

	for _, n := range topo.Nodes {

		if !strings.HasPrefix(n.Name, "n") {
			continue
		}

		dom, err := rvn.DomainInfo("3bed", n.Name)
		if err != nil {
			log.Fatal(err)
		}

		mac := dom.Devices.Interfaces[1].MAC.Address
		mac = strings.Replace(mac, ":", "", -1)

		imap[mac] = "qnode"

	}

	outpath, err := filepath.Abs("../config/files/boss/nodetypes.json")
	if err != nil {
		log.Fatal(err)
	}

	buf, err := json.MarshalIndent(imap, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(outpath, buf, 0644)
	if err != nil {
		log.Fatal(err)
	}

}
