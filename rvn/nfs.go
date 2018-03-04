package rvn

/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * This file implements the nfs functionality of rvn. When rvn creates a
 * virtual machine that mounts folders from the host machine, it does so
 * through NFS. This basically involves setting up the correct exports. That
 * export setup is done here.
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

type Export struct {
	Dir, Subnet string
}

func ExportNFS(topo Topo) error {

	//build the exports table
	table := make(map[string]*Export)
	for _, n := range topo.Nodes {
		for _, m := range n.Mounts {
			table[m.Source] = &Export{
				Dir:    m.Source,
				Subnet: topo.MgmtIp,
			}
		}
	}

	for _, n := range topo.Switches {
		for _, m := range n.Mounts {
			table[m.Source] = &Export{
				Dir:    m.Source,
				Subnet: topo.MgmtIp,
			}
		}
	}

	//flatten table in to a list of exports
	var exports []*Export
	for _, x := range table {
		exports = append(exports, x)
	}

	//run the exports template
	tp_path, err := filepath.Abs("/var/rvn/template/sys.exports")
	if err != nil {
		err = fmt.Errorf("failed to create absolute path for sys.exports - %v", err)
		log.Printf("%v", err)
		return err
	}
	tp, err := template.ParseFiles(tp_path)
	if err != nil {
		err = fmt.Errorf("failed to read sys.exports - %v", err)
		log.Printf("%v", err)
		return err
	}

	wd, err := WkDir()
	if err != nil {
		log.Printf("exportnfs: failed to get working dir")
		return err
	}

	path := fmt.Sprintf("/%s/%s.exports", wd, topo.Name)
	f, err := os.Create(path)
	if err != nil {
		err = fmt.Errorf("failed to create path %s - %v", path, err)
		log.Printf("%v", err)
		return err
	}
	defer f.Close()
	err = tp.Execute(f, exports)
	if err != nil {
		err = fmt.Errorf("failed to execute exports template for %s - %v",
			topo.Name, err)
		log.Printf("%v", err)
		return err
	}

	os.MkdirAll("/etc/exports.d", 0755)
	out, err := exec.Command("cp", path, "/etc/exports.d/").CombinedOutput()
	if err != nil {
		err = fmt.Errorf("unable to create exports directory %s - %v", out, err)
		log.Printf("%v", err)
		return err
	}

	out, err = exec.Command("exportfs", "-ra").CombinedOutput()
	if err != nil {
		err = fmt.Errorf("exportfs failed %s - %v", out, err)
		log.Printf("%v", err)
		return err
	}

	return nil

}

func UnexportNFS(topoName string) error {

	path := fmt.Sprintf("/etc/exports.d/%s.exports", topoName)
	out, err := exec.Command("rm", "-f", path).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error removing exports file %s - %v", path, err)
		return err
	}

	out, err = exec.Command("exportfs", "-ra").CombinedOutput()
	if err != nil {
		err = fmt.Errorf("exportfs failed %s - %v", out, err)
		log.Printf("%v", err)
		return err
	}

	return nil

}
