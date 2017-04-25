#!/bin/sh

set -x
set -e

cd /usr/testbed
mkdir -p obj
cd obj
../src/configure --with-TBDEFS=/opt/deter/defs/defs-vbed-3
cd install
perl ./users-install -b
reboot
