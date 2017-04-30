/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * spine & leaf system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

// define the mounts we will use later in node definitions
deter_mount = {
  'source': '/home/ry/deter',
  'point': '/opt/deter'
};

configMount = (name) => ({
  'source': '/home/ry/raven/models/3bed/config/files/'+name,
  'point': '/tmp/config'
});


// testbed infrastructure nodes
infra = [boss, users] = 
  ['boss', 'users'].map(name => 
    Node(name, 1, [deter_mount, configMount(name)], 'freebsd-11', 'freebsd') 
  );

router = 
  Node('router', 1, [configMount('router')], 'freebsd-11-router', 'freebsd');

// all nodes
nodes = [
  ...Range(3).map(i => Node(`n${i}`, 3, [], 'netboot', 'netboot')),
  ...infra, router,
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
  ...Range(2).map(i => Link(`${infra[i].name}`, 'eth0', 'stem', `swp${i+1}`)),
  Link('router', 'eth0', 'stem', 'swp3'),
  ...Range(3).map(i => Link(`n${i}`, 'eth0', 'stem', `swp${i+4}`, {boot: 1})),
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


