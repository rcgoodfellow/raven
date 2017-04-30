#!/bin/sh

mkdir /tmp/mfs
cd /tmp/mfs

cp /usr/testbed/www/linux-mfs/rootfs.cpio .
/opt/deter/linux-mfs/update_keys.sh \
  rootfs.cpio \
  /usr/testbed/etc/emulab.pem \
  /usr/testbed/etc/client.pem 

cp new-rootfs.cpio /usr/testbed/www/linux-mfs/rootfs.cpio

