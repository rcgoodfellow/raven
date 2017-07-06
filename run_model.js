const fs = require('fs');
const vm = require('vm');

const modeling_src = fs.readFileSync('/usr/local/lib/rvn/modeling.js', 'utf8');
const modeling_script = new vm.Script(modeling_src);
const sandbox = { env: process.env }
const ctx = new vm.createContext(sandbox);
modeling_script.runInContext(ctx);

if(process.argv.length < 3) {
  console.log("usage: run_model <model>");
  process.exit(1);
}
what = process.argv[2];

fs.readFile(what, 'utf8', (err, data) => {
  if (err) throw err;
  run(data);
})

function run(model) {
  const script = new vm.Script(model)
  script.runInContext(ctx);
  console.log(JSON.stringify(ctx.topo, null, 2));
}
