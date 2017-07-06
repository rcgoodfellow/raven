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
  echo -e "${BOLD}go get coffee, this will take many moons $CLEAR\n"

  echo "running users install script"
  rvn ansible  users config/users_install.yml

  echo "rebooting users"
  rvn reboot   users
  sleep 10 #rvn reboot is signal based so need to wait for it to go away

  echo "waiting for users to come back on the network"
  rvn pingwait users

#
# install/setup boss and reboot
#
phase "installing boss"
  echo -e "${BOLD}go get more coffee, this will take many more moons $CLEAR\n"

  echo "running boss install script"
  rvn ansible  boss config/boss_install.yml

  echo "running topology specific setup on boss"
  rvn ansible  boss config/boss_3bed_setup.yml

  echo "rebooting boss"
  rvn reboot   boss
  sleep 20

  echo "waiting for boss to come back on the network"
  rvn pingwait boss

#
# add the nodes to the testbed
#
phase "commissioning"

  echo "rebooting testbed nodes so boss picks them up"
  rvn reboot   n0 n1 n2

  ##TODO you are here --- time to start the deter admin API
  deter_admin="deter-admin `rvn ip boss`"
  echo "waiting for testbed nodes to come up as new nodes"
  cnt=$($deter_admin newnodes | wc -l)
  while [[ "$cnt" -lt "3" ]]; do
    sleep 1
    cnt=$($deter_admin newnodes | wc -l)
  done

  echo "commissioning new nodes"
  rvn ansible  boss config/boss_3bed_commission.yml

  echo "rebooting testbed nodes"
  rvn reboot   n0 n1 n2

  echo "waiting for deter to image and free the testbed nodes"
  cnt=$($deter_admin freenodes | wc -l)
  while [[ "$cnt" -lt "3" ]]; do
    sleep 1
    cnt=$($deter_admin freenodes | wc -l)
  done

#
# run the testsuite from walrus ftw
#
phase "testing"

  echo "launching test suite"
  rvn ansible walrus config/deter_testsuite.yml

  # watch test suite progress
  wtf -collector=`rvn ip walrus` watch config/files/walrus/deter-basic.json

