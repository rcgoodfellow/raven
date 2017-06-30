/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * 5 node raven topology for linux network stack spelunking
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
workspace = '/space/raven/models/spelunk'

nodes = Range(4).map(i => ({
  'name': `n${i}`,
  'image': 'spelnuk-os',
  'os': 'linux',
  'level': 2
}));
nodes[0].level = 1
nodes[1].level = 1
nodes[2].level = 3
nodes[3].level = 3

zwitch = {
  'name': 'nimbus',
  'image': 'spelnuk-os',
  'os': 'linux',
  'level': 2
};

links = Range(4).map(i => Link('nimbus', `eth${i}`, `n${i}`, 'eth0'))

topo = {
  'name': 'spelunk',
  'nodes': nodes,
  'switches': [zwitch],
  'links': links
}

