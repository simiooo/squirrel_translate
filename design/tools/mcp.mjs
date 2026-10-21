import { spawn } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { createInterface } from 'node:readline';
const server = process.env.OPENPENCIL_STDIO;
if (!server) throw new Error('Set OPENPENCIL_STDIO to the installed OpenPencil MCP stdio.mjs path.');
const child = spawn(process.execPath, [server], {
  env: { ...process.env, OPENPENCIL_MCP_ROOT: process.cwd() }, windowsHide: true,
  stdio: ['pipe', 'pipe', 'inherit']
});
let id = 0;
const pending = new Map();
createInterface({ input: child.stdout }).on('line', line => {
  try {
    const msg = JSON.parse(line);
    if (pending.has(msg.id)) {
      const p = pending.get(msg.id); pending.delete(msg.id);
      msg.error ? p.reject(new Error(JSON.stringify(msg.error))) : p.resolve(msg.result);
    }
  } catch (e) { console.error(e.message); }
});
function rpc(method, params) {
  return new Promise((resolve, reject) => {
    const key = ++id; pending.set(key, { resolve, reject });
    child.stdin.write(JSON.stringify({ jsonrpc: '2.0', id: key, method, params }) + '\n');
  });
}
const timeout = setTimeout(() => { console.error('MCP timeout'); child.kill(); process.exit(1); }, 180000);
try {
  await rpc('initialize', { protocolVersion: '2024-11-05', capabilities: {}, clientInfo: { name: 'Squirrel Design', version: '1.0.0' } });
  child.stdin.write(JSON.stringify({ jsonrpc: '2.0', method: 'notifications/initialized' }) + '\n');
  const jobs = process.argv[2] ? JSON.parse(readFileSync(process.argv[2], 'utf8').replace(/^\uFEFF/, '')) : [{ method: 'tools/list' }];
  for (const job of jobs) {
    const result = await rpc(job.method || 'tools/call', job.params || { name: job.name, arguments: job.args || {} });
    if (job.output) writeFileSync(job.output, JSON.stringify(result, null, 2));
    if (job.image) {
      const img = result.content?.find(c => c.type === 'image');
      if (img) writeFileSync(job.image, Buffer.from(img.data, 'base64'));
    }
    console.log(JSON.stringify(job.output || job.image ? { name: job.name, output: job.output, image: job.image, isError: result.isError, text: result.content?.filter(c=>c.type==='text') } : result));
    if (result.isError) throw new Error('Tool failed: ' + job.name);
  }
} finally { clearTimeout(timeout); child.kill(); }
