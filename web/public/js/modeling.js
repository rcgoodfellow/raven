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

Switch = (name, level, mounts) => ({
  'name': name,
  'image': 'cumulus-latest',
  'os': 'linux',
  'level': level,
  'mounts': mounts
});

Node = (name, level, mounts, image, os) => ({
  'name': name,
  'image': image,
  'os': os,
  'level': level,
  'mounts': mounts
});

//generic nic generator
Nic = (n, speed, i) => Array(n).fill({'speed': speed, 'nic': i});

Link = (a, pa, b, pb, props = {}) => ({
    'endpoints': [
      {'name': a, 'port': pa},
      {'name': b, 'port': pb}
    ],
    'name': `${a}_${pa}-${b}_${pb}`,
    'props': props
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

