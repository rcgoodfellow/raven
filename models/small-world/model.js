/*
 * Small-World System
 * ------------------
 *  A raven topology with an IoT device, an Android phone and a server
 *
 */

thing = {
  'name': 'thing',
  'platform': 'arm7',
  'kernel': 'u-boot:a9',
  'os': 'linux',
  'cpu': { 'cores': 1 },
  'memory': { 'capacity': GB(1) }
};

/*
droid = {
  'name': 'droid',
  'platform': 'android',
  'image': 'android-oreo',
  'os': 'linux',
  'cpu': { 'cores': 2 },
  'memory': { 'capacity': GB(2) }
};

server = {
  'name': 'server',
  'platform': 'x86_64',
  'image': 'fedora-27',
  'os': 'linux',
  'cpu': { 'cores': 6 },
  'memory': { 'capacity': GB(12) }
};

sw = {
  'name': 'sw',
  'platform': 'x86_64',
  'image': 'cumulusvx-3.5',
  'os': 'linux'
};
*/

topo = {
  'name': 'small-world',
  'nodes': [thing/*, droid, server*/],
  'switches': [/*sw*/],
  'links': [/*
    Link('thing', 1, 'sw', 1),
    Link('droid', 1, 'sw', 2),
    Link('server', 1, 'sw', 3)
  */],
  'options': {
    'display': 'local'
  }
}

