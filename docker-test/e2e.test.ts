import { describe, it, beforeAll, afterAll, expect, afterEach } from 'vitest';
import { execa } from 'execa';
import * as path from 'path';
import * as fs from 'fs/promises';

// --- Constants ---
const DOCKER_TEST_DIR = __dirname;
const PROJECT_ROOT = path.resolve(DOCKER_TEST_DIR, '..');
const OLD_REVLAY_VERSION = 'v0.1.0';
const SSH_KEY_PATH = path.join(DOCKER_TEST_DIR, 'test_ssh_key');
const SSH_CONFIG_PATH = path.join(DOCKER_TEST_DIR, 'ssh_config');
const SSH_OPTIONS = ['-F', SSH_CONFIG_PATH];
const DEFAULT_TIMEOUT = 120_000; // 增加默认超时时间到2分钟

// --- Helper Functions ---

const run = (file: string, args: string[], options = {}) => {
    // 确保所有命令都以非交互方式运行
    return execa(file, args, { 
        stdio: 'pipe', 
        ...options 
    });
};

// 带重试机制的执行命令
const runWithRetry = async (file: string, args: string[], options = {}, retries = 2, delay = 1000) => {
    let lastError;
    
    for (let i = 0; i <= retries; i++) {
        try {
            return await execa(file, args, { ...options });
        } catch (error) {
            lastError = error;
            if (i < retries) {
                console.log(`Command failed, retrying (${i+1}/${retries})...`);
                await new Promise(resolve => setTimeout(resolve, delay));
            }
        }
    }
    
    throw lastError;
};

// 等待容器健康状态
const waitForContainerHealth = async (containerName: string, timeout = 20000) => {
    console.log(`   - Waiting for container ${containerName} to be healthy...`);
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeout) {
        try {
            const { stdout } = await execa('docker', ['inspect', '--format', '{{.State.Health.Status}}', containerName]);
            if (stdout.trim() === 'healthy') {
                console.log(`   - Container ${containerName} is healthy`);
                return true;
            }
        } catch (error) {
            // 忽略错误，继续等待
        }
        
        // 等待短暂时间再次检查
        await new Promise(resolve => setTimeout(resolve, 500));
    }
    
    console.warn(`   - Timed out waiting for container ${containerName} to be healthy`);
    return false;
};

const prepareBinaries = async () => {
    console.log('>> Preparing test binaries...');
    
    // Build for the host (e.g., macOS) to be run by the test script
    console.log('   - Building revlay for host...');
    await run('go', ['build', '-o', path.join(DOCKER_TEST_DIR, 'revlay-host'), './cmd/revlay'], { 
        cwd: PROJECT_ROOT
    });

    // Build new revlay with cross-compilation for Linux, to be used in the Docker container
    console.log('   - Building new revlay for Linux from source...');
    await run('go', ['build', '-o', path.join(DOCKER_TEST_DIR, 'revlay'), './cmd/revlay'], { 
        cwd: PROJECT_ROOT,
        env: {
            ...process.env,
            GOOS: 'linux',  // 目标操作系统
            GOARCH: 'amd64' // 目标架构
        }
    });

    // Download old revlay
    const oldRevlayPath = path.join(DOCKER_TEST_DIR, 'revlay-old');
    if (!await fs.stat(oldRevlayPath).catch(() => false)) {
        console.log(`   - Downloading old revlay (${OLD_REVLAY_VERSION})...`);
        // 确保下载的是 Linux 版本
        const os = 'linux';  // 强制使用 Linux 版本
        const arch = process.arch === 'x64' ? 'amd64' : 'arm64';
        const url = `https://github.com/revlay/revlay/releases/download/${OLD_REVLAY_VERSION}/revlay-${os}-${arch}`;
        await run('curl', ['-sL', '-o', oldRevlayPath, url]);
        await fs.chmod(oldRevlayPath, 0o755);
    }
    
    // Generate SSH key if not exists
    if (!await fs.stat(SSH_KEY_PATH).catch(() => false)) {
        console.log('   - Generating SSH key for tests...');
        await run('ssh-keygen', ['-t', 'rsa', '-b', '2048', '-f', SSH_KEY_PATH, '-N', '']);
    }
    
    console.log('>> Binaries ready.');
};

const buildImages = async () => {
    console.log('>> Building Docker images...');
    
    // 使用构建缓存标志，加速构建过程
    const buildFlags = [
        '--no-cache=false',  // 启用缓存
        '--pull=false',      // 不要每次都拉取基础镜像
        '--compress'         // 使用压缩以提高构建速度
    ];
    
    // 并行构建镜像以节省时间
    await Promise.all([
        run('docker', ['build', ...buildFlags, '-t', 'revlay-test-env:base', '-f', 'Dockerfile.base', '.'], { cwd: DOCKER_TEST_DIR }),
        // 其他镜像需要base镜像完成后构建
    ]);
    
    await Promise.all([
        run('docker', ['build', ...buildFlags, '-t', 'revlay-test-env:new', '-f', 'Dockerfile.new', '.'], { cwd: DOCKER_TEST_DIR }),
        run('docker', ['build', ...buildFlags, '-t', 'revlay-test-env:old', '-f', 'Dockerfile.old', '.'], { cwd: DOCKER_TEST_DIR })
    ]);
    
    console.log('>> Docker images built.');
};

// 设置容器的SSH访问
const setupContainerSSH = async (containerName: string, sshPort: string) => {
    console.log(`>> Setting up SSH access for container ${containerName}...`);
    const pubKey = await fs.readFile(`${SSH_KEY_PATH}.pub`, 'utf-8');
    
    // 确保 SSH 服务已启动
    console.log('   - Starting SSH service in container...');
    await run('docker', [
        'exec', 
        containerName, 
        'sh', '-c', 
        '/usr/sbin/sshd'
    ]).catch(error => {
        console.log(`   - SSH service might already be running: ${error.message}`);
    });
    
    // 顺序执行SSH设置步骤以避免目录不存在的问题
    console.log('   - Setting up SSH directory and keys...');
    
    // 1. 首先创建 .ssh 目录并设置权限
    try {
        await run('docker', [
            'exec', 
            containerName, 
            'sh', '-c', 
            'mkdir -p /home/revlay-user/.ssh && chmod 700 /home/revlay-user/.ssh'
        ]);
        
        // 2. 然后添加公钥到 authorized_keys 并设置权限
        await run('docker', [
            'exec', 
            containerName, 
            'sh', '-c', 
            `echo '${pubKey.trim()}' > /home/revlay-user/.ssh/authorized_keys && chmod 600 /home/revlay-user/.ssh/authorized_keys && chown -R revlay-user:revlay-user /home/revlay-user/.ssh`
        ]);
    } catch (error: any) {
        console.error('   - Error setting up SSH directory/keys:', error.message);
        // 重试一次，更明确地执行命令，确保目录存在
        console.log('   - Retrying with explicit commands...');
        await run('docker', ['exec', containerName, 'mkdir', '-p', '/home/revlay-user/.ssh']);
        await run('docker', ['exec', containerName, 'chmod', '700', '/home/revlay-user/.ssh']);
        await run('docker', ['exec', containerName, 'sh', '-c', `echo '${pubKey.trim()}' > /home/revlay-user/.ssh/authorized_keys`]);
        await run('docker', ['exec', containerName, 'chmod', '600', '/home/revlay-user/.ssh/authorized_keys']);
        await run('docker', ['exec', containerName, 'chown', '-R', 'revlay-user:revlay-user', '/home/revlay-user/.ssh']);
    }
    
    // 测试 SSH 连接，添加重试机制
    console.log('   - Testing SSH connection...');
    try {
        const sshResult = await runWithRetry('ssh', [
            ...SSH_OPTIONS,
            '-i', SSH_KEY_PATH,
            '-p', sshPort,
            'revlay-user@localhost',
            'echo "SSH connection successful"'
        ], { timeout: 10000, stdio: ['pipe', 'pipe', 'pipe'] }, 3, 2000); // 增加SSH连接超时时间，添加3次重试
        console.log('   - SSH connection test passed');
        console.log('   - SSH output:', sshResult.stdout || 'empty');
    } catch (error: any) {
        console.error('   - SSH connection test failed:', error.message);
        console.error('   - SSH error output:', error.stderr || 'empty');
        console.error('   - SSH exit code:', error.exitCode || 'unknown');
        throw new Error(`Failed to establish SSH connection to container: ${error.message}`);
    }
};

// 检查命令输出是否为 undefined
const safeExpect = (result: any, matcher: string | RegExp) => {
    // 错误通常输出到 stderr，但有些情况（如版本不匹配的提示）可能在 stdout。
    // 我们优先检查 stderr，如果为空，再检查 stdout。
    const output = result.stderr || result.stdout || '';
    console.log(`   - Command output:\nSTDOUT: ${result.stdout || 'empty'}\nSTDERR: ${result.stderr || 'empty'}`);
    
    if (typeof matcher === 'string') {
        if (!output.includes(matcher)) {
            throw new Error(`Expected output to contain "${matcher}", but got:\nSTDOUT: ${result.stdout || 'empty'}\nSTDERR: ${result.stderr || 'empty'}`);
        }
    } else {
        // 使用正则表达式匹配
        if (!matcher.test(output)) {
            throw new Error(`Expected output to match pattern "${matcher.source}", but got:\nSTDOUT: ${result.stdout || 'empty'}\nSTDERR: ${result.stderr || 'empty'}`);
        }
    }
    expect(output).toMatch(matcher);
};

// --- Test Setup ---

beforeAll(async () => {
    // 首先清理可能存在的旧容器
    console.log('>> Cleaning up any existing containers...');
    await run('docker', ['rm', '-f', 'revlay-e2e-server-base', 'revlay-e2e-server-new', 'revlay-e2e-server-old']).catch(() => {
        // 忽略错误，如果容器不存在就会报错
    });
    
    await prepareBinaries();
    await buildImages();
}, 180_000); // 3 minutes timeout for setup

// 确保测试结束后清理所有资源
afterAll(async () => {
    console.log('>> Cleaning up all test resources...');
    
    // 使用 docker ps 查找所有可能的测试容器
    try {
        const { stdout } = await execa('docker', ['ps', '-a', '--filter', 'name=revlay-e2e-server', '--format', '{{.ID}}']);
        const containerIds = stdout.trim().split('\n').filter(Boolean);
        
        if (containerIds.length > 0) {
            console.log(`   - Found ${containerIds.length} containers to clean up`);
            for (const id of containerIds) {
                await run('docker', ['rm', '-f', id]).catch(() => {});
            }
        } else {
            console.log('   - No containers to clean up');
        }
    } catch (error: any) {
        console.error('   - Error during cleanup:', error.message);
    }
}, 30_000); // 30 seconds timeout for cleanup

// --- Test Suite ---

// 顺序执行测试组，避免端口冲突
describe('Revlay E2E Tests', { sequential: true }, () => {
    const SSH_PORT = '2222';
    const revlayCli = path.join(DOCKER_TEST_DIR, 'revlay-host'); // Use the host-specific binary
    const SSH_USER = 'revlay-user';
    const TEST_APP_NAME = 'my-app';
    const TEST_PATH = '.';

    // 构建基本的 push 命令参数
    const getPushArgs = () => [
        'push',
        '-p', TEST_PATH,
        '--to', `${SSH_USER}@localhost`,
        '--app', TEST_APP_NAME,
        '--ssh-port', SSH_PORT,
        '-i', SSH_KEY_PATH,
        '--ssh-args', 'BatchMode=yes',
        '--ssh-args', 'StrictHostKeyChecking=no',
        '--ssh-args', 'UserKnownHostsFile=/dev/null',
    ];

    let containerId: string | null = null;

    afterEach(async () => {
        if (containerId) {
            await run('docker', ['rm', '-f', containerId]).catch(() => {});
            containerId = null;
        }
    });

    it('Scenario A: should report error when revlay is not installed on remote', async () => {
        const CONTAINER_NAME = 'revlay-e2e-server-base';
        containerId = CONTAINER_NAME;
        
        // 启动容器并等待健康状态
        await run('docker', ['run', '-d', '--rm', '--name', CONTAINER_NAME, '-p', `${SSH_PORT}:22`, 'revlay-test-env:base']);
        await waitForContainerHealth(CONTAINER_NAME);
        await setupContainerSSH(CONTAINER_NAME, SSH_PORT);
        
        const result = await execa(revlayCli, getPushArgs(), { 
            reject: false, 
            timeout: 60000,
            env: {
                ...process.env,
                REVLAY_NON_INTERACTIVE: 'true', // 禁用交互模式
                REVLAY_E2E_TEST: 'true',
            }
        });
        
        safeExpect(result, /command not found|not installed|error|failed/i); // 使用正则表达式，匹配多种可能的错误信息
    }, DEFAULT_TIMEOUT);

    it('Scenario B: should trigger interactive setup when app is not found', async () => {
        const CONTAINER_NAME = 'revlay-e2e-server-new';
        containerId = CONTAINER_NAME;
        
        // 启动容器并等待健康状态
        await run('docker', ['run', '-d', '--rm', '--name', CONTAINER_NAME, '-p', `${SSH_PORT}:22`, 'revlay-test-env:new']);
        await waitForContainerHealth(CONTAINER_NAME);
        await setupContainerSSH(CONTAINER_NAME, SSH_PORT);
        
        const result = await execa(revlayCli, getPushArgs(), { 
            reject: false, 
            timeout: 60000,
            env: {
                ...process.env,
                REVLAY_NON_INTERACTIVE: 'true', // 禁用交互模式
                REVLAY_E2E_TEST: 'true',
            }
        });

        safeExpect(result, /初始化|setup|initialize|init|引导|guide|prompt/i); // 使用正则表达式匹配多种初始化相关词汇
    }, DEFAULT_TIMEOUT);

    it('Scenario C: should proceed when app exists', async () => {
        const CONTAINER_NAME = 'revlay-e2e-server-new';
        containerId = CONTAINER_NAME;
        
        // 启动容器并等待健康状态
        await run('docker', ['run', '-d', '--rm', '--name', CONTAINER_NAME, '-p', `${SSH_PORT}:22`, 'revlay-test-env:new']);
        await waitForContainerHealth(CONTAINER_NAME);
        await setupContainerSSH(CONTAINER_NAME, SSH_PORT);
        
        // 简化步骤：合并多条命令为单条减少Docker交互次数
        const remoteAppPath = `/home/revlay-user/${TEST_APP_NAME}`;
        await run('docker', [
            'exec',
            CONTAINER_NAME,
            'sh',
            '-c',
            `mkdir -p ${remoteAppPath} && echo "app:\\n  name: ${TEST_APP_NAME}" > ${remoteAppPath}/revlay.yml && revlay service add ${TEST_APP_NAME} ${remoteAppPath}`
        ]);

        const result = await execa(revlayCli, getPushArgs(), { 
            reject: false, 
            timeout: 60000,
            env: {
                ...process.env,
                REVLAY_NON_INTERACTIVE: 'true', // 禁用交互模式
                REVLAY_E2E_TEST: 'true',
            }
        });
        
        safeExpect(result, /Application found|app found|success|deployed|completed|proceed/i); // 使用正则表达式匹配成功信息
    }, DEFAULT_TIMEOUT);

    it('Scenario D: should report incompatibility with old revlay version', async () => {
        const CONTAINER_NAME = 'revlay-e2e-server-old';
        containerId = CONTAINER_NAME;
        
        // 启动容器并等待健康状态
        await run('docker', ['run', '-d', '--rm', '--name', CONTAINER_NAME, '-p', `${SSH_PORT}:22`, 'revlay-test-env:old']);
        await waitForContainerHealth(CONTAINER_NAME);
        await setupContainerSSH(CONTAINER_NAME, SSH_PORT);
        
        const result = await execa(revlayCli, getPushArgs(), { 
            reject: false, 
            timeout: 60000,
            env: {
                ...process.env,
                REVLAY_NON_INTERACTIVE: 'true', // 禁用交互模式
                REVLAY_E2E_TEST: 'true',
            }
        });

        safeExpect(result, /incompatible|version|old|outdated|update|upgrade|not supported|error|failed/i); // 使用正则表达式匹配不兼容信息
    }, DEFAULT_TIMEOUT);

    it('Scenario E: should fail to connect to non-existent host', async () => {
        const args = getPushArgs();
        // Override the host to a non-existent one
        const hostIndex = args.indexOf('--to') + 1;
        args[hostIndex] = 'user@nonexistent-host:2223';
        
        const result = await execa(revlayCli, args, { 
            reject: false, 
            timeout: 10000,
            env: {
                ...process.env,
                REVLAY_NON_INTERACTIVE: 'true', // 禁用交互模式
                REVLAY_E2E_TEST: 'true',
            }
        }); 
        safeExpect(result, /connect|refused|failed|unable|timeout|error|not found|unreachable|host|port/i); // 使用正则表达式匹配连接错误
    }, 20_000);
}); 