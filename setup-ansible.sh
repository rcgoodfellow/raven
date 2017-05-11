#!/bin/bash

dist=`lsb_release -i | awk '{print $3}'`

if [[ "$dist" -eq "Debian" ]]; then
	echo "deb http://ppa.launchpad.net/ansible/ansible/ubuntu trusty main" | sudo tee /etc/apt/sources.list.d/ansible.list
else
	sudo apt-get install -y software-properties-common
	sudo apt-add-repository -y ppa:ansible/ansible
fi

sudo apt-get update
sudo apt-get install -y ansible
exit


