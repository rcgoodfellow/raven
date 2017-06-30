package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/rcgoodfellow/raven/rvn"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 4 {
		usage()
	}

	switch os.Args[1] {
	case "reboot":
		doReboot(os.Args[2:])
	default:
		usage()
	}
}

func doReboot(args []string) {

	if len(args) < 2 {
		usage()
	}

	rr := rvn.RebootRequest{
		Topo:  args[0],
		Nodes: args[1:],
	}

	js, err := json.MarshalIndent(rr, "", "  ")
	if err != nil {
		log.Println(err)
	}
	buf := bytes.NewBuffer(js)

	resp, err := http.Post("http://localhost:9000/rvn-reboot", "text/json", buf)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		} else {
			log.Println(string(body))
		}
	}

}

func usage() {
	s := red("usage:\n")
	s += fmt.Sprintf("\t%s %s system node-1 ... node-n",
		blue("rvn"), green("reboot"))

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
