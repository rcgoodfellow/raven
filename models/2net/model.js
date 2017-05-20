/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * 2 node system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
workspace = '/space/raven/models/2net'
zwitch = {
  'name': 'nimbus',
  'image': 'cumulus-latest',
  'os': 'linux',
  'level': 1,
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
  'level': 2
}));

links = Range(2).map(i => Link(`n${i}`, 'eth0', 'nimbus', `swp${i}`));

topo = {
  'name': '2net',
  'nodes': nodes,
  'switches': [zwitch],
  'links': links
};
