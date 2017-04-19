package rvn

import (
	xlibvirt "github.com/libvirt/libvirt-go-xml"
	"strings"
)

func runHooks(dom *xlibvirt.Domain) {
	if strings.Contains(dom.Devices.Disks[0].Source.File, "cumulus") {
		cumulusHook(dom)
	}
}

func cumulusHook(dom *xlibvirt.Domain) {
	//do cumulus specific configuration here
}
