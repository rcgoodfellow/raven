package rvn

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

type TpData struct {
	Host Host
	NFS  string
}

var testData = []struct {
	Object   *TpData
	Expected []string
}{
	{
		Object: &TpData{
			Host: Host{
				Name: "smooth-llama",
				OS:   "debian-stretch",
				Mounts: []Mount{
					Mount{
						Point:  "/opt/smooth",
						Source: "/llama-src",
					},
				},
			},
			NFS: "192.168.254.253",
		},
		Expected: []string{
			`---
- hosts: all
  become: true

  tasks:
    - name: determine os
      command: uname -s
      register: ostype

    - name: copy utils
      copy:
        src: /var/rvn/util/iamme-linux
        dest: /usr/local/bin/iamme
        mode: "a+x"
      when: ostype.stdout == "Linux"
    
    - name: copy utils
      copy:
        src: /var/rvn/util/iamme-freebsd
        dest: /usr/local/bin/iamme
        mode: "a+x"
      when: ostype.stdout == "FreeBSD"

    - name: set hostname
      hostname:
        name: smooth-llama 

    - name: put hostname in /etc/hosts
      lineinfile:
        name: /etc/hosts
        line: '127.0.0.1    smooth-llama'

    - name: update libvirt dns
      command: /usr/local/bin/iamme eth0 192.168.254.253
      when: ostype.stdout == "Linux"
    
    #
    #- name: update libvirt dns
    #  command: /usr/local/bin/iamme vtnet0 192.168.254.253
    #  when: ostype.stdout == "FreeBSD"


    - name: mount /opt/smooth
      mount:
        name: /opt/smooth
        src: 192.168.254.253:/llama-src
        opts: rw,soft
        fstype: nfs
        state: mounted

`,
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
