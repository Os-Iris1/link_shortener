import assert from 'node:assert/strict';
import { once } from 'node:events';
import http from 'node:http';
import { mkdtemp, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';
import { build, createServer, preview } from 'vite';
import config from './vite.config.js';

const root = fileURLToPath(new URL('.', import.meta.url));

test('Vite proxies short-link queries in dev and preview', { timeout: 60000 }, async (t) => {
  const received = [];
  const backend = http.createServer((request, response) => {
    received.push(request.url);
    if (request.url.startsWith('/api/')) {
      response.writeHead(201, { 'Content-Type': 'application/json' });
      response.end('{"short_url":"https://short.example/abcdefghijkl"}');
    } else if (request.url === '/healthz') {
      response.end('ok');
    } else {
      response.writeHead(302, { Location: 'https://example.com/target?q=1#fragment' });
      response.end();
    }
  });
  backend.listen(0, '127.0.0.1');
  await once(backend, 'listening');
  t.after(() => new Promise((resolve) => { backend.closeAllConnections(); backend.close(resolve); }));
  const target = `http://127.0.0.1:${backend.address().port}`;
  const directory = await mkdtemp(path.join(os.tmpdir(), 'shorter-vite-test-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const common = {
    ...config, root, configFile: false, envFile: false, logLevel: 'silent',
    cacheDir: path.join(directory, 'cache'),
    build: { outDir: path.join(directory, 'dist'), emptyOutDir: true },
  };
  await build(common);
  for (const mode of ['server', 'preview']) {
    await t.test(mode, async () => {
      // Only replace destinations; route matching comes from the real config.
      const proxy = Object.fromEntries(Object.entries(config[mode].proxy).map(([route, options]) => [
        route, typeof options === 'string' ? target : { ...options, target },
      ]));
      const settings = { ...config[mode], host: '127.0.0.1', port: 0, proxy };
      const instance = mode === 'server'
        ? await createServer({ ...common, server: settings })
        : await preview({ ...common, preview: settings });
      try {
        if (mode === 'server') await instance.listen();
        const port = instance.httpServer.address().port;
        const request = (url, method = 'GET') => new Promise((resolve, reject) => {
          const req = http.request({ hostname: '127.0.0.1', port, path: url, method }, (response) => {
            let body = '';
            response.setEncoding('utf8');
            response.on('data', (chunk) => { body += chunk; });
            response.on('end', () => resolve({ status: response.statusCode, headers: response.headers, body }));
          });
          req.setTimeout(5000, () => req.destroy(new Error('HTTP timeout')));
          req.on('error', reject);
          req.end();
        });
        for (const suffix of ['', '?', '?utm_source=test', '?a=1&b=2', '?q=%D1%8F&x=%23%2F']) {
          for (const method of ['GET', 'HEAD']) {
            const url = '/abcdefghijkl' + suffix;
            const result = await request(url, method);
            assert.equal(result.status, 302, `${mode} ${method} ${url}`);
            assert.equal(result.headers.location, 'https://example.com/target?q=1#fragment');
            // Vite's HTTP proxy normalizes a bare '?' to no query; values stay exact.
            assert.equal(received.at(-1), suffix === '?' ? '/abcdefghijkl' : url, 'query values must arrive unchanged');
            if (method === 'HEAD') assert.equal(result.body, '');
          }
        }
        for (const url of ['/abcdefghijkl/extra?x=1', '/abcdefghijklm?x=1', '/abcdefghijk?x=1', '/']) {
          const count = received.length;
          const result = await request(url);
          assert.equal(received.length, count, `${url} must not reach backend`);
          assert.match(result.headers['content-type'], /text\/html/);
          assert.match(result.body, /id="root"/);
        }
        assert.equal((await request('/api/links', 'POST')).status, 201);
        assert.equal((await request('/healthz')).body, 'ok');
      } finally {
        if (mode === 'server') await instance.close();
        else await new Promise((resolve) => { instance.httpServer.closeAllConnections(); instance.httpServer.close(resolve); });
      }
    });
  }
});
