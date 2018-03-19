/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * 2 node system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

controller = {
  'name': 'control',
  'image': 'debian-stretch',
  'os': 'linux',
  'cpu': {
    'cores': 4
  },
  'memory': {
    'capacity': GB(4)
  }
}

walrus = {
  'name': 'walrus',
  'image': 'debian-stretch',
  'os': 'linux',
  'cpu': {
    'cores': 1
  },
  'memory': {
    'capacity': GB(2)
  },
  'mounts': [
    { 'source': env.PWD+'/walrustf', 'point': '/opt/walrus'},
    { 'source': env.PWD+'/config/files/walrus', 'point': '/tmp/config' }
  ]
}

zwitch = {
  'name': 'nimbus',
  'image': 'cumulusvx-3.5',
  'os': 'linux',
  'mounts': [
    { 'source': env.PWD+'/config/files/nimbus', 'point': '/tmp/config' }
  ]
};

nodes = Range(2).map(i => ({
  'name': `n${i}`,
  'image': 'debian-stretch',
  'os': 'linux',
  'cpu': {
    'cores': 4
  },
  'memory': {
    'capacity': GB(6)
  },
  'mounts': [
    { 'source': env.PWD+'/walrustf', 'point': '/opt/walrus'},
    { 'source': env.PWD+'/config/files/node', 'point': '/tmp/config' }
  ]
}));

links = [
  Link('walrus', 0, 'nimbus', 1),
  Link('control', 0, 'nimbus', 2),
  ...Range(2).map(i => Link(`n${i}`, 0, 'nimbus', i+3)),
]

topo = {
  'name': '2net',
  'nodes':[controller, walrus, ...nodes],
  'switches': [zwitch],
  'links': links
};
