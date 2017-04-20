var rspace = null;
var status_ = null;

function getStatus(category, what) {
  if(status_ != null) {
    if(what in status_[category]) {
      return status_[category][what];
    }
  }
  return '?';
}

function showResourceModel(tbdata) {
  var c1 = document.getElementById('tb-net');
  var tbopts = {
    nodes: {
      font: {
        size: 25,
        color: '#222'
      },
      borderWidth: 1
    },
    edges: {
      width: 2
    },
    layout: {
      improvedLayout: false,
      hierarchical: {
        direction: "UD"
      }
    },

  };
  net = new vis.Network(c1, tbdata, tbopts);
  net.on("click", function(params) {
    var xi = document.getElementById('tb-info');
    xi.innerHTML = ""
    console.log(params);
    if(params.nodes.length > 0) {
      let name = params.nodes[0];
      let node = rspace.nodes.find(x => x.name == name);
      if(node !== undefined) {
        console.log(node);
        node['status'] = getStatus('nodes', name);
        xi.innerHTML = '<pre>'+JSON.stringify(node, null, 2)+'</pre>';
      } 
      else{
        node = rspace.switches.find(x => x.name == name);
        if(node !== undefined) {
          console.log(node);
          node['status'] = getStatus('switches', name);
          xi.innerHTML = '<pre>'+JSON.stringify(node, null, 2)+'</pre>';
        } 
      }
    }
    else if(params.edges.length > 0) {
      let name = params.edges[0];
      let link = rspace.links.find(x => x.name == name);
      if(link!== undefined) {
        console.log(link);
        link['status'] = getStatus('links', name);
        xi.innerHTML = '<pre>'+JSON.stringify(link, null, 2)+'</pre>';
      } 
    }
  });
}


function getTopo(src) {
 eval(src);
 return topo; 
}

function showTopo(topo) {
  rspace = topo;
  var viz = rvnToVis(topo);
  showResourceModel(viz);
}

function handleUpload(evt) {
  var file = evt.files[0];
  var reader = new FileReader();
  reader.onload = function(theFile) {
    console.log('rspace');
    var src = reader.result;
    console.log(src);
    var topo = getTopo(src);
    rspace = topo;
    console.log(topo);
    var viz = rvnToVis(topo);
    console.log(viz);
    showResourceModel(viz);
  };
  reader.readAsText(file);
}

function push() {
  console.log('push');
  console.log(rspace);

  var xhr = new XMLHttpRequest();
  xhr.open('POST', '/rvn-push');
  xhr.setRequestHeader('Content-Type', 'application/json; charset=UTF-8');
  xhr.send(JSON.stringify(rspace));
  xhr.onloadend = function() {
    console.log('push request completed');
  }
}

function status() {
  console.log('status');
  $.get("/rvn-status?topo="+rspace.name, function(data) {
    console.log(data);
    status_ = data;
    var xi = document.getElementById('tb-info');
    xi.innerHTML = '<pre>'+JSON.stringify(data, null, 2)+'</pre>';
  });
}

function destroy() {
  console.log('destroy');
  $.get("/rvn-destroy?topo="+rspace.name, function(data) {
    console.log(data);
  });
}

function launch() {
  console.log('launch');
  $.get("/rvn-launch?topo="+rspace.name, function(data) {
    console.log(data);
  });
}

function configure() {
  console.log('configure');
  $.get("/rvn-configure?topo="+rspace.name, function(data) {
    console.log(data);
  });
}
