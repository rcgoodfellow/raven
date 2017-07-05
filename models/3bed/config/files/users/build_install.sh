#!/bin/sh

set -x
set -e

cd /usr/testbed
mkdir -p obj
cd obj
../src/configure --with-TBDEFS=/opt/deter/defs/defs-vbed-3
cd install
perl ./users-install -b

chmod 0711 /etc/ssh/external_keys
chown -R rvn:rvn /etc/ssh/external_keys/rvn
chmod 0700 /etc/ssh/external_keys/rvn
chmod 0600 /etc/ssh/external_keys/rvn/authorized_keys

