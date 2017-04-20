/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * spine & leaf system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

Switch = (name, level) => ({
  'name': name,
  'os': 'cumulus-latest',
  'level': level
});

Node = (name, level, mounts) => ({
  'name': name,
  'os': 'debian-stretch',
  'level': level,
  'mounts': mounts
});



infra = ['boss', 'users', 'router'];
nodes = [
  ...Range(3).map(i => Node(`n${i}`, 3)),
  ...infra.map(n => Node(n, 1, [{
      'source': '/home/ry/deter', 
      'point': '/usr/testbed/src'
    }])),
  Node('walrus', 2, [{
    'source': '/home/ry/deter/walrustf',
    'point': '/opt/walrus'
  }])
];

switches = [
  Switch('stem', 2),
  Switch('leaf', 4)
];

links = [
  ...Range(3).map(i => Link(`${infra[i]}`, 'eth0', 'stem', `swp${i}`)),
  ...Range(3).map(i => Link(`n${i}`, 'eth0', 'stem', `swp${i+3}`)),
  ...Range(3).map(i => Link(`n${i}`, 'eth0', 'leaf', `swp${i}`)),
  Link('walrus', 'eth0', 'stem', 'swp7')
];

topo = {
  'name': '3bed',
  'nodes': nodes,
  'switches': switches,
  'links': links
};


