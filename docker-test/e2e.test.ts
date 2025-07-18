import { describe, it, beforeAll, afterAll, expect, afterEach } from 'vitest';
import { execa } from 'execa';
import * as path from 'path';
import * as fs from 'fs/promises';

const DOCKER_TEST_DIR = __dirname;
const PROJECT_ROOT = path.resolve(DOCKER_TEST_DIR, '..');
const OLD_REVLAY_VERSION = 'v0.1.0';

// --- Helper Functions ---

const run = (file: string, args: string[], options = {}) => {
    return execa(file, args, { stdio: 'inherit', ...options });
};

const prepareBinaries = async () => {
    console.log('>> Preparing test binaries...');
    
    // Build new revlay
    console.log('   - Building new revlay from source...');
    await run('go', ['build', '-o', path.join(DOCKER_TEST_DIR, 'revlay'), './cmd/revlay'], { cwd: PROJECT_ROOT });

    // Download old revlay
    const oldRevlayPath = path.join(DOCKER_TEST_DIR, 'revlay-old');
    if (!await fs.stat(oldRevlayPath).catch(() => false)) {
        console.log(`   - Downloading old revlay (${OLD_REVLAY_VERSION})...`);
        const os = process.platform === 'darwin' ? 'darwin' : 'linux';
        const arch = process.arch === 'x64' ? 'amd64' : 'arm64';
        const url = `https://github.com/revlay/revlay/releases/download/${OLD_REVLAY_VERSION}/revlay-${os}-${arch}`;
        await run('curl', ['-sL', '-o', oldRevlayPath, url]);
        await fs.chmod(oldRevlayPath, 0o755);
    }
    console.log('>> Binaries ready.');
};

const buildImages = async () => {
    console.log('>> Building Docker images...');
    await run('docker', ['build', '-t', 'revlay-test-env:base', '-f', 'Dockerfile.base', '.'], { cwd: DOCKER_TEST_DIR });
    await run('docker', ['build', '-t', 'revlay-test-env:new', '-f', 'Dockerfile.new', '.'], { cwd: DOCKER_TEST_DIR });
    await run('docker', ['build', '-t', 'revlay-test-env:old', '-f', 'Dockerfile.old', '.'], { cwd: DOCKER_TEST_DIR });
    console.log('>> Docker images built.');
};

// --- Test Setup ---

beforeAll(async () => {
    await prepareBinaries();
    await buildImages();
}, 120_000); // 2 minutes timeout for setup

// --- Test Suite ---

describe('Revlay E2E Tests', () => {

    const CONTAINER_NAME = 'revlay-e2e-server';
    const SSH_PORT = '2222';
    const revlayCli = path.join(DOCKER_TEST_DIR, 'revlay');

    let containerId: string | null = null;

    afterEach(async () => {
        if (containerId) {
            await run('docker', ['rm', '-f', CONTAINER_NAME]);
            containerId = null;
        }
    });

    it('Scenario A: should report error when revlay is not installed on remote', async () => {
        await run('docker', ['run', '-d', '--rm', '-p', `${SSH_PORT}:22`, '--name', CONTAINER_NAME, 'revlay-test-env:base']);
        
        // This is a placeholder for the actual test command.
        // You would run `revlay push` here and assert on its output.
        const { stderr } = await execa(revlayCli, ['push', /* ...args */], { reject: false });
        
        expect(stderr).toContain('revlay command not found on remote host');
    }, 30_000);

    it('Scenario B: should trigger interactive setup when app is not found', async () => {
        await run('docker', ['run', '-d', '--rm', '-p', `${SSH_PORT}:22`, '--name', CONTAINER_NAME, 'revlay-test-env:new']);
        // Placeholder
        const { stdout } = await execa(revlayCli, ['push', /* ...args */], { reject: false });
        expect(stdout).toContain('是否立即进行初始化引导');
    });

    it('Scenario C: should proceed when app exists', async () => {
        await run('docker', ['run', '-d', '--rm', '-p', `${SSH_PORT}:22`, '--name', CONTAINER_NAME, 'revlay-test-env:new']);
        await run('docker', ['exec', CONTAINER_NAME, 'revlay', 'service', 'add', '--name', 'my-app', '--from', 'http://example.com']);

        // Placeholder
        const { stdout } = await execa(revlayCli, ['push', /* ...args */], { reject: false });
        expect(stdout).toContain('Application found, proceeding with deployment...');
    });

    it('Scenario D: should report incompatibility with old revlay version', async () => {
        await run('docker', ['run', '-d', '--rm', '-p', `${SSH_PORT}:22`, '--name', CONTAINER_NAME, 'revlay-test-env:old']);
        // Placeholder
        const { stderr } = await execa(revlayCli, ['push', /* ...args */], { reject: false });
        expect(stderr).toContain('Remote revlay version is incompatible');
    });

    it('Scenario E: should fail to connect to non-existent host', async () => {
        // No container is started for this test
        const { stderr } = await execa(revlayCli, ['push', '--ssh-port', '9999' /* ...other args */], { reject: false });
        expect(stderr).toMatch(/ssh: connect to host localhost port 9999: Connection refused/);
    });

}); 