///
/// Util
///

function Range(n) {
  return [...Array(n).keys()]
}

function flatmap(f) {
  return [].concat.apply([], this.map(f));
}

function mod(n, m) {
  return ((n % m) + m) % m;
}

///
/// Modeling Convinence
///

//generic nic generator
Nic = (n, speed, i) => Array(n).fill({'speed': speed, 'nic': i});

Link = (a, pa, b, pb, speed) => ({
    'endpoints': [
      {'name': a, 'port': pa},
      {'name': b, 'port': pb}
    ],
    'name': `${a}_${pa}-${b}_${pb}`,
    'capacity': speed
  }
);

Image = (name, arch, version) => ({
  'name': name, 
  'arch': arch,
  'version': version
});

Topo = (nodes, images, links, switches) => ({
  'nodes': nodes,
  'images': images,
  'links': links,
  'switches': switches
})

