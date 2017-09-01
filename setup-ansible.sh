#!/bin/bash

set -x
set -e


dist=`lsb_release -i | awk '{print $3}'`

if [[ "$dist" -eq "Debian" ]]; then
  sudo apt install dirmngr
  sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 93C4A3FD7BB9C367
  echo "deb http://ppa.launchpad.net/ansible/ansible/ubuntu trusty main" | sudo tee /etc/apt/sources.list.d/ansible.list
else
  sudo apt install -y software-properties-common
  sudo apt-add-repository -y ppa:ansible/ansible
fi

sudo apt update
sudo apt install -y ansible
exit


