/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * 2 node system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

zwitch = {
  'name': 'nimbus',
  'os': 'cumulus-latest',
  'level': 1
};

nodes = Range(2).map(i => ({
  'name': `n${i}`,
  'os': 'debian-stretch',
  'level': 2
}));

nodes[0]['mounts'] = [
  {
    'source': '/home/ry/deter', 
    'point': '/opt/deter'
  }
];

links = Range(2).map(i => Link(`n${i}`, 'eth0', 'nimbus', `swp${i}`));

topo = {
  'name': '2net',
  'nodes': nodes,
  'switches': [zwitch],
  'links': links
};
