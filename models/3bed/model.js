/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * spine & leaf system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

deter_mount = {
  'source': '/home/ry/deter',
  'point': '/opt/deter'
};

configMount = (name) => ({
  'source': '/home/ry/raven/models/3bed/config/files/'+name,
  'point': '/tmp/config'
});


infra = [boss, users, router] = 
  ['boss', 'users', 'router'].map(name => 
    Node(name, 1, [deter_mount, configMount(name)], 'freebsd-11', 'freebsd') 
  ) 


nodes = [
  ...Range(3).map(i => Node(`n${i}`, 3, [], 'debian-stretch', 'linux')),
  ...infra,
  Node('walrus', 
    2, [{
      'source': '/home/ry/deter/walrustf',
      'point': '/opt/walrus'
    }],
    'debian-stretch', 'linux'
  )
];


switches = [
  Switch('stem', 2, [deter_mount, configMount('stem')]),
  Switch('leaf', 4, [deter_mount, configMount('leaf')])
];

links = [
  ...Range(3).map(i => Link(`${infra[i].name}`, 'eth0', 'stem', `swp${i+1}`)),
  ...Range(3).map(i => Link(`n${i}`, 'eth0', 'stem', `swp${i+4}`)),
  ...Range(3).map(i => Link(`n${i}`, 'eth0', 'leaf', `swp${i+1}`)),
  Link('walrus', 'eth0', 'stem', 'swp7'),
  Link('stem', 'swp8', 'leaf', 'swp4')
];

topo = {
  'name': '3bed',
  'nodes': nodes,
  'switches': switches,
  'links': links
};


