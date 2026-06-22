// VoiceAssistant 生产环境前端静态服务器
// - 托管 Vue SPA 静态文件
// - /api/ 反向代理到后端（含 WebSocket 升级）
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';

const PORT = 2501;
const DIST = './dist';
const BACKEND = process.env.BACKEND_URL || 'http://backend:2500';

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.ico': 'image/x-icon',
  '.json': 'application/json; charset=utf-8',
  '.woff2': 'font/woff2',
};

function staticFile(filePath) {
  try {
    const data = fs.readFileSync(filePath);
    const ext = path.extname(filePath);
    return { data, contentType: MIME[ext] || 'application/octet-stream' };
  } catch {
    return null;
  }
}

function proxyReq(req, res) {
  const url = new URL(req.url, BACKEND);
  const opts = {
    hostname: url.hostname,
    port: url.port,
    path: url.pathname + url.search,
    method: req.method,
    headers: { ...req.headers, host: url.host },
  };
  const proxy = http.request(opts, (pRes) => {
    res.writeHead(pRes.statusCode, pRes.headers);
    pRes.pipe(res);
  });
  proxy.on('error', () => {
    res.writeHead(502);
    res.end('Bad Gateway');
  });
  req.pipe(proxy);
}

function upgradeProxy(req, socket, head) {
  const url = new URL(req.url, BACKEND);
  const opts = {
    hostname: url.hostname,
    port: url.port,
    path: url.pathname + url.search,
    method: req.method,
    headers: { ...req.headers, host: url.host },
  };
  const proxy = http.request(opts);
  proxy.on('upgrade', (pRes, pSocket, pHead) => {
    socket.write('HTTP/1.1 101 Switching Protocols\r\n');
    for (const [k, v] of Object.entries(pRes.headers)) {
      socket.write(`${k}: ${v}\r\n`);
    }
    socket.write('\r\n');
    socket.pipe(pSocket);
    pSocket.pipe(socket);
  });
  proxy.on('error', () => socket.destroy());
  proxy.end();
}

// 需代理到后端的非 API 路径
const BACKEND_ROUTES = ['/health'];

const server = http.createServer((req, res) => {
  // API + 后端独有路由代理
  if (req.url.startsWith('/api/') || BACKEND_ROUTES.some(r => req.url.startsWith(r))) {
    return proxyReq(req, res);
  }

  // 静态文件
  let filePath = path.join(DIST, req.url === '/' ? 'index.html' : req.url);
  let file = staticFile(filePath);
  if (file) {
    res.writeHead(200, { 'Content-Type': file.contentType });
    return res.end(file.data);
  }

  // SPA 回退
  file = staticFile(path.join(DIST, 'index.html'));
  if (file) {
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    return res.end(file.data);
  }

  res.writeHead(404);
  res.end('Not Found');
});

// WebSocket 代理
server.on('upgrade', (req, socket, head) => {
  if (req.url.startsWith('/api/')) {
    return upgradeProxy(req, socket, head);
  }
  socket.destroy();
});

server.listen(PORT, () => {
  console.log(`Frontend server listening on port ${PORT}, proxy -> ${BACKEND}`);
});
