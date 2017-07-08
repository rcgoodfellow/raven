#!/bin/bash

export WKDIR=`pwd`

if [[ ! -d src ]]; then
mkdir src
fi

cd src

if [[ ! -d agx ]]; then
git clone git@github.com:rcgoodfellow/agx
fi

if [[ ! -d switch-drivers ]]; then
git clone git@github.com:deter-project/switch-drivers
fi

if [[ ! -d netlink ]]; then
git clone git@github.com:rcgoodfellow/netlink
fi

if [[ ! -d walrustf ]]; then
git clone git@github.com:rcgoodfellow/walrustf
fi

export AGXDIR=`pwd`/agx
export SWITCHDIR=`pwd`/switch-drivers
export NETLINKDIR=`pwd`/netlink
export WALRUSDIR=`pwd`/walrustf

cd $WKDIR

