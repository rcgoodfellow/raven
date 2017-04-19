
function rvnToVis(input) {

  nodes = input.nodes.map(x => ({
    id: x.name, 
    label: x.name, 
    shape: 'box',
    level: x.level,
    data: { kind: 'node' }
  }));

  switches = input.switches.map(x => ({
    id: x.name, 
    label: x.name, 
    shape: 'box',
    color: {
      background: '#83c985',
      border: '#2f6a31'
    },
    level: x.level,
    data: { kind: 'net'}
  }));

  links = input.links.map(x => ({
    id: x.name,
    from: x.endpoints[0].name,
    to: x.endpoints[1].name
  }));

  elements = {
    nodes: [...nodes, ...switches],
    edges: links
  }

  return elements;

}
