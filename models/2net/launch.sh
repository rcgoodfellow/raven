#!/bin/bash

set -e

BOLD="\e[1m"
BLUE="\e[34m"
CLEAR="\e[0m"

function phase() {
echo -e "$BOLD
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\r$1
$CLEAR"
}

# launch the system and wait till it is up

phase "Fetching walrus"
  if [[ ! -d walrustf ]]; then
    git clone https://github.com/rcgoodfellow/walrustf
  fi

phase "Building"
  echo "clearing out any artifacts from previous runs"
  sudo -E rvn destroy
  echo "building system"
  sudo -E rvn build

phase "Deploying"
  echo "launching vms"
  sudo -E rvn deploy
  echo "waiting for vms to come on network"
  sudo -E rvn pingwait control walrus nimbus n0 n1

phase "Configuring"
  sudo -E rvn configure

phase "Testing"
  echo "launching tests"
  sudo -E rvn ansible walrus config/run_tests.yml
  wtf -collector=`rvn ip walrus` watch config/files/walrus/tests.json
