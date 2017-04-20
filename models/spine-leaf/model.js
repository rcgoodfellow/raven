/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * spine & leaf system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

Switch = (name, level) => ({
  'name': name,
  'os': 'cumulus-latest',
  'level': level
});

Node = (name, level) => ({
  'name': name,
  'os': 'debian-stretch',
  'level': level
});

spine_leaf = (s, l) => Link(
  `spine${ s[0] }`, `swp${ s[1] }`, 
   `leaf${ l[0] }`, `swp${ l[1] }`
);

leaf_node =  (l, n) => Link(
  `leaf${ l[0] }`, `swp${ l[1] }`, 
     `n${ n[0] }`, `eth${ n[1] }`
);

stem_node = (s, n) => Link(
  `stem${ s[0] }`, `swp${ s[1] }`, 
     `n${ n[0] }`, `eth${ n[1] }`
);

control = Switch('stem0', 4)
boss = Node('boss', 5)
boss['mounts'] = [
  ['/home/ry/deter', 'deter'],
]
users = Node('users', 5)
router = Node('router', 5)

spine = Range(2).map(i => Switch(`spine${i}`, 1));
leaf  = Range(4).map(i => Switch(`leaf${i}`, 2));
node  = Range(8).map(i => Node(`n${i}`, 3));

trunk  = Range(8).map(i => spine_leaf([i%2  , i/2|0], 
                                      [i/2|0, i%2  ]));

stem   = Range(8).map(i => leaf_node( [i/2|0, 2+(i%2)], 
                                      [i    , 1      ]));

branch = Range(8).map(i => stem_node( [0, i], 
                                      [i, 0]));

blink = Link('stem0', 'swp8', 'boss', 'eth0');
ulink = Link('stem0', 'swp9', 'users', 'eth0');
rlink = Link('stem0', 'swp10', 'router', 'eth0');
slink = Link('stem0', 'swp11', 'spine0', 'swp5');

topo = {
  'name': 'spine-leaf',
  'nodes': [...node, boss, users, router],
  'switches': [control, ...spine, ...leaf],
  'links': [...branch, ...trunk, ...stem, blink, ulink, rlink, slink]
};

