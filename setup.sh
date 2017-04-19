#!/bin/sh

#set up directories
sudo mkdir -p /var/rvn/img
sudo touch /var/rvn/run

#install base images
cd /var/rvn/img
sudo wget http://mirror.deterlab.net/rvn/cumulus-latest.qcow2
sudo wget http://mirror.deterlab.net/rvn/debian-stretch.qcow2

#install ssh keys
mkdir -p ~/.ssh
cd ~/.ssh
wget http://mirror.deterlab.net/rvn/rvn
wget http://mirror.deterlab.net/rvn/rvn.pub
