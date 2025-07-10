# Revlay CLI - Basic Usage Guide

## Overview

Revlay is a modern, dependency-free deployment tool for Linux servers that provides:
- Atomic deployments using symlink switching
- Zero-downtime and short-downtime deployment modes
- Easy rollbacks
- Structured directory management (releases/shared/current)
- YAML configuration support
- Multi-language support (Chinese/English)

## Language Support

Revlay supports both Chinese and English:

```bash
# Use Chinese (default)
revlay status

# Use English
revlay --lang=en status

# Set environment variable
export REVLAY_LANG=en
revlay status
```

## Getting Started

### 1. Initialize a new project

```bash
# Interactive initialization (Chinese)
revlay init

# Interactive initialization (English)
revlay --lang=en init

# Or with command line flags
revlay init --name myapp --host server.example.com --user deploy --path /opt/myapp
```

This creates a `revlay.yml` configuration file in your current directory.

### 2. Configure your project

Edit the generated `revlay.yml` file:

```yaml
app:
  name: myapp
  repository: https://github.com/user/myapp.git
  branch: main
  keep_releases: 5

server:
  host: server.example.com
  user: deploy
  port: 22
  key_file: ~/.ssh/id_rsa

deploy:
  path: /opt/myapp
  mode: zero_downtime  # or short_downtime
  shared_paths:
    - storage/logs
    - storage/uploads
  environment:
    NODE_ENV: production

service:
  command: "cd ${RELEASE_PATH} && PORT=${PORT} node server.js"
  port: 8080
  alt_port: 8081
  health_check: "/health"
  restart_delay: 5
  graceful_timeout: 30

hooks:
  pre_deploy: []
  post_deploy:
    - systemctl reload nginx
  pre_rollback: []
  post_rollback:
    - systemctl reload nginx
```

### 3. Deploy your application

```bash
# Deploy with auto-generated timestamp
revlay deploy

# Deploy with custom release name
revlay deploy v1.0.0

# Dry run to see what would happen (explains deployment plan)
revlay deploy --dry-run
```

### 4. Manage releases

```bash
# List all releases
revlay releases

# Check deployment status
revlay status

# Rollback to previous release
revlay rollback

# Rollback to specific release
revlay rollback v1.0.0
```

## Deployment Modes

Revlay supports two deployment strategies:

### Zero Downtime Deployment (default)

Uses blue-green deployment with port switching:
- Starts new service on alternative port
- Performs health checks
- Switches traffic via load balancer
- Gracefully shuts down old service

**Best for:**
- Stateless applications
- Applications with external storage (Redis, database)
- Cloud-native applications
- Applications with load balancers

### Short Downtime Deployment

Uses traditional stop-update-start approach:
- Stops current service
- Updates symlink to new release
- Starts new service

**Best for:**
- Applications with file locking
- Applications with database locking
- Single-instance applications
- Applications that load global state at startup

## Configuration Reference

### App Section
- `name`: Application name
- `repository`: Git repository URL (future use)
- `branch`: Git branch to deploy (future use)
- `keep_releases`: Number of releases to keep

### Server Section
- `host`: Server hostname or IP
- `user`: SSH username
- `port`: SSH port (default: 22)
- `password`: SSH password (optional)
- `key_file`: SSH private key file (optional)

### Deploy Section
- `path`: Base deployment path on server
- `mode`: Deployment mode (`zero_downtime` or `short_downtime`)
- `shared_paths`: Directories to share between releases
- `environment`: Environment variables

### Service Section (for zero_downtime mode)
- `command`: Service start command with placeholders
- `port`: Primary service port
- `alt_port`: Alternative port for blue-green deployment
- `health_check`: Health check URL path
- `restart_delay`: Delay between retries (seconds)
- `graceful_timeout`: Graceful shutdown timeout (seconds)

### Hooks Section
- `pre_deploy`: Commands to run before deployment
- `post_deploy`: Commands to run after deployment
- `pre_rollback`: Commands to run before rollback
- `post_rollback`: Commands to run after rollback

## Dry Run Functionality

The `--dry-run` flag shows what would happen without making changes:

```bash
revlay deploy --dry-run
```

This displays:
- Deployment plan and configuration
- Directory structure to be created
- Shared paths to be linked
- Hooks to be executed
- Deployment mode specific settings
- Service ports and health check configuration

## Directory Structure

Revlay creates and manages the following directory structure on your server:

```
/opt/myapp/
├── releases/
│   ├── 20240101-120000/  # Release directories
│   ├── 20240101-130000/
│   └── v1.0.0/
├── shared/               # Shared files between releases
│   ├── storage/
│   │   ├── logs/
│   │   └── uploads/
│   └── config/
└── current -> releases/20240101-130000  # Symlink to active release
```

## Key Features

### Atomic Deployments
- Uses symlink switching for instant, atomic deployments
- Automatic rollback on failure
- Zero corruption during deployment

### Shared Resources
- Configure shared directories and files
- Automatically linked to each release
- Persistent across deployments

### Release Management
- Automatic cleanup of old releases
- Configurable retention policy
- Easy rollback to any previous release

### Deployment Hooks
- Pre/post deployment scripts
- Pre/post rollback scripts
- Environment variable substitution

### Port Management (Zero Downtime)
- Blue-green deployment with port switching
- Automatic service health checks
- Load balancer integration support

## SSH Authentication

Revlay supports multiple SSH authentication methods:

1. **SSH Key (recommended):**
   ```yaml
   server:
     key_file: ~/.ssh/id_rsa
   ```

2. **Password:**
   ```yaml
   server:
     password: your-password
   ```

3. **SSH Agent:** Automatically used if available

## Examples

### Zero Downtime Web API
```yaml
app:
  name: api
  keep_releases: 5
server:
  host: api.example.com
  user: deploy
deploy:
  path: /opt/api
  mode: zero_downtime
  shared_paths:
    - storage/logs
service:
  command: "cd ${RELEASE_PATH} && PORT=${PORT} node server.js"
  port: 8080
  alt_port: 8081
  health_check: "/health"
hooks:
  post_deploy:
    - "/opt/api/scripts/update_nginx.sh"
```

### Short Downtime Traditional App
```yaml
app:
  name: webapp
  keep_releases: 3
server:
  host: web.example.com
  user: deploy
deploy:
  path: /var/www/webapp
  mode: short_downtime
  shared_paths:
    - storage/logs
    - public/uploads
    - data/database.db
service:
  command: "systemctl restart webapp"
  graceful_timeout: 30
hooks:
  pre_deploy:
    - "systemctl stop webapp"
  post_deploy:
    - "systemctl start webapp"
    - "systemctl reload nginx"
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `revlay init` | Initialize a new project |
| `revlay deploy` | Deploy a new release |
| `revlay deploy --dry-run` | Preview deployment plan |
| `revlay rollback` | Rollback to previous release |
| `revlay releases` | List all releases |
| `revlay status` | Show deployment status |
| `revlay --lang=en <cmd>` | Use English language |
| `revlay --help` | Show help information |

## Best Practices

1. **Choose the right deployment mode**
   - Use zero-downtime for stateless applications
   - Use short-downtime for applications with file/database locks

2. **Configure health checks** for zero-downtime deployments

3. **Use SSH keys** instead of passwords for authentication

4. **Test deployments** with `--dry-run` flag first

5. **Monitor releases** with `revlay status`

6. **Keep backups** of your configuration files

7. **Use shared paths** for persistent data

8. **Configure hooks** for application-specific tasks

9. **Set appropriate keep_releases** limit for your disk space

10. **Plan for port conflicts** in zero-downtime mode

## Troubleshooting

### SSH Connection Issues
- Verify server hostname and port
- Check SSH key permissions (should be 600)
- Ensure SSH key is added to server's authorized_keys
- Test SSH connection manually: `ssh user@host`

### Deployment Failures
- Check server disk space
- Verify deployment path permissions
- Review hook commands for errors
- Check `revlay status` for detailed information

### Port Conflicts (Zero Downtime)
- Ensure primary and alternative ports are different
- Check no other services are using configured ports
- Verify load balancer configuration

### Service Health Checks
- Test health check endpoint manually: `curl http://localhost:8080/health`
- Verify service starts correctly on alternative port
- Check service logs for startup errors

### Permission Issues
- Ensure deploy user has write access to deployment path
- Check shared directory permissions
- Verify hook commands can be executed by deploy user

## Advanced Topics

For detailed information about deployment modes, port conflict resolution, and handling database/file locking, see [DEPLOYMENT_MODES.md](DEPLOYMENT_MODES.md).