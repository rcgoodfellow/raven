#!/bin/bash

set -e

trinket=('/' '-' '\\' '|')

deter_admin="deter-admin `rvn ip boss`"
echo "waiting for testbed nodes to come up as free nodes"
i=0
cnt=$($deter_admin freenodes | wc -l)
while [[ "$cnt" -lt "3" ]]; do
  sleep 1
  cnt=$($deter_admin freenodes | wc -l)
  x=$((i % 4))
  printf "$cnt free nodes in testbed ${trinket[$x]}\r"
  i=$((i + 1))
done
