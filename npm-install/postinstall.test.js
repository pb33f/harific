import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, readFile, copyFile, symlink, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer } from 'node:http';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import * as tar from 'tar';

const run = promisify(execFile);

test('installer downloads and extracts the release archive', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'harific-installer-'));
    let server;
    try {
        await mkdir(join(dir, 'fixture'));
        await writeFile(join(dir, 'fixture', 'harific'), 'fixture binary');
        await tar.c({ gzip: true, file: join(dir, 'release.tar.gz'), cwd: join(dir, 'fixture') }, ['harific']);
        const archive = await readFile(join(dir, 'release.tar.gz'));
        server = createServer((req, res) => res.end(archive));
        await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
        const root = fileURLToPath(new URL('../', import.meta.url));
        await symlink(join(root, 'node_modules'), join(dir, 'node_modules'), 'junction');
        await copyFile(new URL('./postinstall.js', import.meta.url), join(dir, 'postinstall.js'));
        await writeFile(join(dir, 'package.json'), JSON.stringify({ type: 'module', version: '1.0.0' }));
        await writeFile(join(dir, 'config.js'), `export const CONFIG = {name:'harific',path:'./bin',url:'http://127.0.0.1:${server.address().port}/release.tar.gz'}; export const ARCH_MAPPING = {}; export const PLATFORM_MAPPING = {};`);
        await run(process.execPath, ['postinstall.js'], { cwd: dir });
        assert.equal(await readFile(join(dir, 'bin', 'harific'), 'utf8'), 'fixture binary');
        await assert.rejects(readFile(join(dir, 'downloaded.tar.gz')), { code: 'ENOENT' });
    } finally {
        if (server) await new Promise(resolve => server.close(resolve));
        await rm(dir, { recursive: true, force: true });
    }
});
