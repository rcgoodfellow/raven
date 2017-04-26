#!/bin/sh

set -x
set -e

cd /usr/testbed/obj
../src/configure --with-TBDEFS=/opt/deter/defs/defs-vbed-3
gmake boss-install-force
