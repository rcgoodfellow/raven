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

#
# build the system description and allocate resources for it
#
phase "building"

  echo "clearing out any artifacts from previous test runs"
  rvn destroy

  echo "building system"
  rvn build

#
# launch the system and wait users, boss and the primary swtich are up
#
phase "deploying"

  echo "lanuching virtual system"
  rvn deploy

  echo "waiting for vms to come on network"
  rvn pingwait users boss router stem leaf walrus

  echo "configuring vms"
  rvn configure

#
# install users and reboot
#
phase "installing users"
  echo "$BOLD go get coffee, this will take many moons $CLEAR"

  echo "running users install script"
  rvn ansible  users config/users_install.yml

  echo "rebooting users"
  rvn reboot   users

  echo "waiting for users to come back on the network"
  rvn pingwait users

#
# install/setup boss and reboot
#
phase "installing boss"
  echo "$BOLD go get more coffee, this will take many moons $CLEAR"

  echo "running boss install script"
  rvn ansible  boss config/boss_install.yml

  echo "running topology specific setup on boss"
  rvn ansible  boss config/boss_3bed_setup.yml

  echo "rebooting boss"
  rvn reboot   boss

  echo "waiting for boss to come back on the network"
  rvn pingwait boss

#
# add the nodes to the testbed
#
chatper "commissioning"

  echo "rebooting testbed nodes so boss picks them up"
  rvn reboot   n0 n1 n2

  echo "waiting for testbed nodes to come up as new nodes"
  cnt=$(deter newnodes count)
  while [[ "$cnt" -lt "3" ]]; do
    sleep 1
    cnt=$(deter nodes count --where state=pxewait)
  done

  echo "commissioning new nodes"
  rvn ansible  boss boss_3bed_commission.yml

  echo "rebooting testbed nodes"
  rvn reboot   n0 n1 n2

  echo "waiting for deter to image and free the testbed nodes"
  cnt=$(deter nodes count --where state=pxewait)
  while [[ "$cnt" -lt "3" ]]; do
    sleep 1
    cnt=$(deter nodes count --where state=pxewait)
  done

#
# run the testsuite from walrus ftw
#
chapter "testing"

  echo "launching test suite"
  rvn ansible walrus deter_testsuite.yml

  # watch test suite progress
  wtf -collector=`rvn ip walrus` watch config/files/walrus/tests.json

