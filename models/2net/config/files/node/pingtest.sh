#!/usr/bin/env bash

usage() {
  echo "usage: "
  echo "  pingtest address test-id"
}

if [[ "$#" -ne 2 ]]; then
  usage
  exit 1
fi

target=$1
testid=$2
hostname=`hostname`
wtf=/opt/walrus/bash/walrus.sh

echo "starting ping test"
trap "exit" INT
while true; do
  ping -q -w 1 $target &> /dev/null
  if [[ "$?" -ne 0 ]]; then
    $wtf walrus warning $testid $hostname 0 ping-failed
    printf ". "
  else
    $wtf walrus ok $testid $hostname 0 ping-success
    printf "+"
  fi
  i=$((i+1))
done
