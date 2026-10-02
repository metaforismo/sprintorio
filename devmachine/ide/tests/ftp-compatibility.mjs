// Run against code-server's installed dependency tree:
// node ftp-compatibility.mjs /usr/lib/code-server
// For optional FTPS, generate a short-lived fixture certificate with openssl:
// openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=localhost \
//   -addext 'subjectAltName=IP:127.0.0.1,DNS:localhost' -keyout key.pem -out cert.pem
// Set SPRINTORIO_FTP_TEST_CERT=cert.pem and SPRINTORIO_FTP_TEST_KEY=key.pem.
import assert from 'node:assert/strict';
import net from 'node:net';
import tls from 'node:tls';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { resolve } from 'node:path';

const packageRoot = resolve(process.argv[2] || '/usr/lib/code-server');
const require = createRequire(`${packageRoot}/package.json`);
const { getUri } = require('get-uri');
const getUriRequire = createRequire(require.resolve('get-uri'));
assert.equal(require('get-uri/package.json').version, '6.0.5');
assert.equal(getUriRequire('basic-ftp/package.json').version, '6.2.1');
const content = 'Sprintorio FTP compatibility\n';
const modified = '20261003120000';
// Optional local FTPS fixture credentials. Trust is scoped to get-uri's client,
// without changing process-wide CA settings or disabling certificate validation.
const tlsOptions = process.env.SPRINTORIO_FTP_TEST_CERT && process.env.SPRINTORIO_FTP_TEST_KEY
  ? { cert: readFileSync(process.env.SPRINTORIO_FTP_TEST_CERT), key: readFileSync(process.env.SPRINTORIO_FTP_TEST_KEY) }
  : undefined;

async function listen(server) {
  await new Promise((accept, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', accept);
  });
  return server.address().port;
}

async function fixture({ passive = 'EPSV', mdtm = true, separateHost = false, secure = false } = {}) {
  const sockets = new Set();
  const dataServers = new Set();
  const commands = [];
  let dataConnections = 0;
  const track = socket => {
    sockets.add(socket);
    socket.once('close', () => sockets.delete(socket));
    return socket;
  };
  const createServer = handler => secure ? tls.createServer(tlsOptions, handler) : net.createServer(handler);
  const server = createServer(socket => {
    track(socket);
    socket.on('error', () => {});
    socket.write('220 Sprintorio local compatibility fixture\r\n');
    let input = '';
    let dataSocket;
    let chain = Promise.resolve();
    const reply = text => socket.write(`${text}\r\n`);
    const openData = async command => {
      const dataServer = secure ? tls.createServer(tlsOptions) : net.createServer();
      dataServers.add(dataServer);
      dataSocket = new Promise(accept => dataServer.once(secure ? 'secureConnection' : 'connection', connection => {
        dataConnections++;
        track(connection);
        connection.on('error', () => {});
        accept(connection);
      }));
      const port = await listen(dataServer);
      if (command === 'EPSV') reply(`229 Entering Extended Passive Mode (|||${port}|)`);
      else reply(`227 Entering Passive Mode (${separateHost ? '192,0,2,1' : '127,0,0,1'},${port >> 8},${port & 255})`);
    };
    const transfer = async payload => {
      reply('150 Opening data connection');
      const data = await dataSocket;
      await new Promise(accept => data.end(payload, accept));
      reply('226 Transfer complete');
    };
    const handle = async line => {
      const [command, ...args] = line.split(' ');
      commands.push(command);
      switch (command) {
        case 'USER': reply('331 Password required'); break;
        case 'PASS': reply('230 Logged in'); break;
        case 'FEAT': reply('211-Features\r\n MLST type*;size*;modify*;\r\n211 End'); break;
        case 'OPTS': case 'TYPE': case 'STRU': case 'PBSZ': case 'PROT': reply('200 OK'); break;
        case 'PWD': reply('257 "/"'); break;
        case 'MDTM':
          if (args.join(' ').includes('missing')) reply('550 File not found');
          else reply(mdtm ? `213 ${modified}` : '502 MDTM not supported');
          break;
        case 'EPSV':
          if (passive === 'EPSV') await openData(command);
          else reply('502 EPSV not supported');
          break;
        case 'PASV': await openData(command); break;
        case 'MLSD':
          await transfer(`type=file;size=${Buffer.byteLength(content)};modify=${modified}; fixture.txt\r\n`);
          break;
        case 'RETR': await transfer(content); break;
        case 'QUIT': reply('221 Goodbye'); socket.end(); break;
        default: reply('502 Command not implemented');
      }
    };
    socket.on('data', chunk => {
      input += chunk.toString();
      let newline;
      while ((newline = input.indexOf('\r\n')) >= 0) {
        const line = input.slice(0, newline);
        input = input.slice(newline + 2);
        chain = chain.then(() => handle(line)).catch(error => {
          socket.destroy(error);
        });
      }
    });
  });
  const port = await listen(server);
  return {
    url: `ftp://fixture:password@127.0.0.1:${port}/fixture.txt`,
    commands,
    clientOptions: secure ? { secure: 'implicit', secureOptions: { ca: tlsOptions.cert } } : {},
    get dataConnections() { return dataConnections; },
    async close() {
      for (const socket of sockets) socket.destroy();
      await Promise.all([...dataServers, server].map(listener => new Promise(accept => listener.close(accept))));
    },
  };
}

async function read(stream) {
  let result = '';
  for await (const chunk of stream) result += chunk.toString();
  return result;
}

async function run(name, options, verify) {
  const ftp = await fixture(options);
  try {
    await verify(ftp);
    console.log(`PASS ${name}`);
  } finally {
    await ftp.close();
  }
}

const timeout = setTimeout(() => {
  console.error('FTP compatibility fixture timed out');
  process.exit(1);
}, 15000);
try {
  for (const passive of ['EPSV', 'PASV']) {
    await run(`${passive} download, metadata, cache and missing file`, { passive }, async ftp => {
      const stream = await getUri(ftp.url);
      assert.equal(await read(stream), content);
      assert.equal(stream.lastModified.toISOString(), '2026-10-03T12:00:00.000Z');
      assert.ok(ftp.commands.includes(passive));
      await assert.rejects(getUri(ftp.url, { cache: stream }), { code: 'ENOTMODIFIED' });
      await assert.rejects(getUri(ftp.url.replace('fixture.txt', 'missing.txt')), { code: 'ENOTFOUND' });
    });
    await run(`${passive} MDTM fallback to MLSD`, { passive, mdtm: false }, async ftp => {
      const stream = await getUri(ftp.url);
      assert.equal(await read(stream), content);
      assert.equal(stream.lastModified.toISOString(), '2026-10-03T12:00:00.000Z');
      assert.ok(ftp.commands.includes('MLSD'));
    });
  }
  // The advertised address is documentation-only. The client must reject it
  // before connecting; no listener or connection is created at that address.
  await run('separate PASV host refused before data connection', { passive: 'PASV', mdtm: false, separateHost: true }, async ftp => {
    await assert.rejects(getUri(ftp.url), /PASV returned another host/);
    assert.equal(ftp.dataConnections, 0);
    assert.ok(!ftp.commands.includes('RETR'));
  });
  if (tlsOptions) {
    await run('implicit FTPS with verified fixture certificate', { secure: true }, async ftp => {
      await assert.rejects(getUri(ftp.url, { secure: 'implicit' }), { code: 'DEPTH_ZERO_SELF_SIGNED_CERT' });
      const stream = await getUri(ftp.url, ftp.clientOptions);
      assert.equal(await read(stream), content);
      assert.equal(stream.lastModified.toISOString(), '2026-10-03T12:00:00.000Z');
      assert.ok(ftp.commands.includes('PROT'));
    });
  } else {
    console.log('NOT RUN optional FTPS: set SPRINTORIO_FTP_TEST_CERT and SPRINTORIO_FTP_TEST_KEY');
  }
  console.log(`PASS get-uri 6.0.5 / basic-ftp 6.2.1 on ${process.version}`);
} finally {
  clearTimeout(timeout);
}
