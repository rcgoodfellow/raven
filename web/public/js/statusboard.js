
function runStatusBoard() {
  window.setInterval(function() {
    refreshStatusBoard()
  }, 1000);
}

function refreshStatusBoard() {
  $.get("/rvn-status?topo="+topo+"&fragment=true", function(data) {
    $("#status-container").html(data);
  });
}
