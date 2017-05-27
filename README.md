![Raven](doc/raven.png)
<br />
# Raven
**R**y's **A**pparatus for **V**irtual **E**ncodable **N**etworks

Raven is a tool for rapidly designing, visualizing, deploying and managing virtual networks. Raven networks are:
- designed programatically through a javascript API
- visualized and managed through a web interface
- materialized and deployed by a libvirt enabled backend with Cumulus VX virtual switches

Here is an example of a network model

```javascript
workspace = '/space/raven/models/2net'

controller = {
  'name': 'control', 'image': 'debian-stretch', 'os': 'linux', 'level': 1,
  'mounts': [{ 'source': '/space/switch-drivers', 'point': '/opt/switch-drivers'}]
}

walrus = {
  'name': 'walrus', 'image': 'debian-stretch', 'os': 'linux', 'level': 2,
  'mounts': [{ 'source': '/space/walrustf', 'point': '/opt/walrus'}]
}

zwitch = {
  'name': 'nimbus', 'image': 'cumulus-latest', 'os': 'linux', 'level': 2,
  'mounts': [{ 'source': '/space/netlink', 'point': '/opt/netlink' }]
};

nodes = Range(2).map(i => ({
  'name': `n${i}`, 'image': 'debian-stretch', 'os': 'linux', 'level': 3
}));

links = [
  Link('walrus', 'eth0', 'nimbus', 'swp1'),
  Link('control', 'eth0', 'nimbus', 'swp2'),
  ...Range(2).map(i => Link(`n${i}`, 'eth0', 'nimbus', `swp${i+3}`)),
]

topo = {
  'name': '2net',
  'nodes':[controller, walrus, ...nodes],
  'switches': [zwitch],
  'links': links
};
```
This file looks like the following when uploaded through the web interface
<br />
<br />
<img src='http://mirror.deterlab.net/rvn/doc/2net-web.png' width="600" />

Use the push, destroy,, launch, mount and configure buttons to realize, configure and work with your code inside a virtual realization of the environment model. 

<!--
See [this article](http://dev.goodwu.net/distributed-systems/testing/networking/infrastructure/2017/05/26/distributed-walrus.html)for a more complete tutorial.
-->

## Getting started
I have tested Raven on Debian-Stretch and Ubuntu 16.04. Contributions to support other distros welcome!

### Installing

```shell
git clone git@github.com:rcgoodfellow/raven
cd raven
./setup-ansible.sh
ansible-playbook setup.yml

```

### Tinkering
First start the raven application (you must be root due to the way we use libvirt)

```shell
sudo su
cd $GOPATH:/src/github.com/rcgoodfellow/raven/web
revel run
```

Then open up a web browser using a path to a raven directoy. Raven itself comes with a few. Try 
```
http://localhost:9000/?dir=/space/raven/models/2net
```
for starters. To access virtual machines you can use the `rvn-ssh` command for example

```shell
rvn-ssh 2net n0
```
Raven also comes with the `rvn-ansible` command for launching ansible playbooks against virtual machines in a convinient ad-hoc nature.

