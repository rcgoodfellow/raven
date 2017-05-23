/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * 2 node system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
workspace = '/space/raven/models/2net'

controller = {
  'name': 'control',
  'image': 'debian-stretch',
  'os': 'linux',
  'level': 1,
  'mounts': [
    { 'source': '/space/switch-drivers',          'point': '/opt/switch-drivers'},
    { 'source': workspace+'/config/files/controller', 'point': '/tmp/config' }
  ]
}

walrus = {
  'name': 'walrus',
  'image': 'debian-stretch',
  'os': 'linux',
  'level': 2,
  'mounts': [
    { 'source': '/space/walrustf',                  'point': '/opt/walrus'},
    { 'source': workspace+'/config/files/walrus', 'point': '/tmp/config' }
  ]
}

zwitch = {
  'name': 'nimbus',
  'image': 'cumulus-latest',
  'os': 'linux',
  'level': 2,
  'mounts': [
    { 'source': '/space/agx',                     'point': '/opt/agx' },
    { 'source': '/space/netlink',                 'point': '/opt/netlink' },
    { 'source': '/space/switch-drivers',          'point': '/opt/switch-drivers'},
    { 'source': workspace+'/config/files/nimbus', 'point': '/tmp/config' }
  ]
};

nodes = Range(2).map(i => ({
  'name': `n${i}`,
  'image': 'debian-stretch',
  'os': 'linux',
  'level': 3
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
