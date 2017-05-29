package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/rcgoodfellow/raven/rvn"
	"github.com/revel/revel"
	"html/template"
	"io/ioutil"
	"log"
	"os/exec"
)

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	title := "rvn"
	moreStyles := []string{
		"https://cdnjs.cloudflare.com/ajax/libs/vis/4.17.0/vis.min.css",
		"/public/css/rvn.css",
	}
	moreScripts := []string{
		"https://cdnjs.cloudflare.com/ajax/libs/vis/4.17.0/vis.min.js",
		"/public/js/modeling.js",
		"/public/js/rvn-vjs.js",
		"/public/js/tb-to-visjs.js",
	}

	dir := c.Params.Query.Get("dir")
	if len(dir) == 0 {
		c.Response.Status = 400
		return c.RenderText("please provide a working directory")
	}

	out, err := exec.Command(
		"nodejs",
		"/usr/local/lib/rvn/run_model.js",
		dir+"/model.js",
	).CombinedOutput()

	if err != nil {
		log.Printf("error running model int %s - %s - %v", dir, out, err)
		c.Response.Status = 400
		return c.RenderText("error running model %s", out)
	}

	topo := rvn.ReadTopo(out)
	topo.Dir = dir
	data, _ := json.MarshalIndent(topo, "", "  ")
	model := template.JS("var topo = " + string(data) + ";")

	return c.Render(title, moreStyles, moreScripts, model)
}

func (c App) Push() revel.Result {
	var topo rvn.Topo
	body, _ := ioutil.ReadAll(c.Request.Body)
	if len(body) == 0 {
		c.Response.Status = 400
		return c.RenderText("bad argument")
	}

	json.Unmarshal(body, &topo)
	rvn.Create(topo)
	c.Response.Status = 200
	return c.RenderText("ok")
}

func (c App) Mount() revel.Result {
	var topo rvn.Topo
	body, _ := ioutil.ReadAll(c.Request.Body)
	if len(body) == 0 {
		c.Response.Status = 400
		return c.RenderText("bad argument")
	}
	json.Unmarshal(body, &topo)
	rvn.ExportNFS(topo)
	rvn.GenConfigAll(topo)
	c.Response.Status = 200
	return c.RenderText("ok")
}

func (c App) Status() revel.Result {
	moreStyles := []string{
		"/public/css/status.css",
	}

	moreScripts := []string{
		"/public/js/statusboard.js",
	}

	topo := c.Params.Query.Get("topo")
	fragment := c.Params.Query.Get("fragment")
	//log.Printf("status: topo=%s fragment=%s", topo, fragment)
	if len(topo) == 0 {
		c.Response.Status = 400
		return c.RenderText("bad argument")
	}
	status := rvn.Status(topo)
	nodes := status["nodes"]
	switches := status["switches"]

	meta := template.JS(fmt.Sprintf("var topo = '%s';", topo))

	if fragment == "true" {
		c.ViewArgs["nodes"] = nodes
		c.ViewArgs["switches"] = switches
		return c.RenderTemplate("statusboard.html")
	} else {
		return c.Render(meta, nodes, switches, moreStyles, moreScripts)
	}
}

func (c App) Destroy() revel.Result {
	topo := c.Params.Query.Get("topo")
	log.Printf("destroy: topo=%s", topo)
	if len(topo) == 0 {
		c.Response.Status = 400
		return c.RenderText("bad argument")
	}

	rvn.Destroy(topo)
	c.Response.Status = 200
	return c.RenderText("ok")
}

func (c App) Launch() revel.Result {
	topo := c.Params.Query.Get("topo")
	log.Printf("launch: topo=%s", topo)
	if len(topo) == 0 {
		c.Response.Status = 400
		return c.RenderText("bad argument")
	}

	errors := rvn.Launch(topo)
	if len(errors) == 0 {
		c.Response.Status = 200
		return c.RenderText("ok")
	} else {
		c.Response.Status = 200
		return c.RenderJSON(errors)
	}
}

func (c App) Configure() revel.Result {
	topo := c.Params.Query.Get("topo")
	log.Printf("configure: topo=%s", topo)
	if len(topo) == 0 {
		c.Response.Status = 400
		return c.RenderText("bad argument")
	}

	go rvn.Configure(topo, true)
	c.Response.Status = 200
	return c.RenderText("ok")
}
