package rvn

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

var testData = []struct {
	Object   *Host
	Expected []string
}{
	{
		Object: &Host{
			Name: "smooth-llama",
			OS:   "debian-stretch",
			Mounts: []Mount{
				Mount{
					Point:  "/opt/smooth",
					Source: "/llama-src",
					Tag:    "smooth-llama_opt_smooth",
				},
			},
		},
		Expected: []string{
			`---`,
			`- hosts: all`,
			`  become: true`,
			``,
			`  tasks:`,
			`    - name: set hostname`,
			`      hostname:`,
			`        name: smooth-llama`,
			``,
			`    - name: put hostname in /etc/hosts`,
			`      lineinfile:`,
			`        name: /etc/hosts`,
			`        line: '127.0.0.1    smooth-llama'`,
			``,
			``,
			`    - name: mount /opt/smooth`,
			`      mount:`,
			`        name: /opt/smooth`,
			`        src: smooth-llama_opt_smooth`,
			`        fstype: 9p`,
			`        opts: trans=virtio,rw`,
			`        state: mounted`,
			``,
		},
	},
}

func TestConfig(t *testing.T) {
	tp, err := template.ParseFiles("config.yml")
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range testData {
		var doc bytes.Buffer
		err := tp.Execute(&doc, test.Object)
		if err != nil {
			t.Fatal(err)
		}

		expect := strings.Join(test.Expected, "\n")

		if doc.String() != expect {
			t.Fatal(
				"Bad config:\n~", doc.String(), "~\n does not match\n~", expect, "~\n")
		}
	}
}
