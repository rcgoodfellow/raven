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

phase "Building"
  echo "clearing out any artifacts from previous runs"
  rvn destroy
  echo "building system"
  rvn build

phase "Deploying"
  echo "launching vms"
  rvn deploy
  echo "waiting for vms to come on network"
  rvn pingwait control walrus nimbus n0 n1

phase "Configuring"
  rvn configure

phase "Testing"
  echo "launching tests"
  rvn ansible walrus config/run_tests.yml
  wtf -collector=`rvn ip walrus` watch config/files/walrus/tests.json
