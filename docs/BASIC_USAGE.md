# Revlay CLI - Basic Usage Guide

## Overview

Revlay is a modern, dependency-free deployment tool for Linux servers that provides:
- Atomic deployments using symlink switching
- Zero-downtime deployments
- Easy rollbacks
- Structured directory management (releases/shared/current)
- YAML configuration support

## Getting Started

### 1. Initialize a new project

```bash
# Interactive initialization
revlay init

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
  shared_paths:
    - storage/logs
    - storage/uploads
  environment:
    NODE_ENV: production

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

# Dry run to see what would happen
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
- No downtime during deployment
- Automatic rollback on failure

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

### Basic Web Application
```yaml
app:
  name: webapp
  keep_releases: 5
server:
  host: web.example.com
  user: deploy
deploy:
  path: /var/www/webapp
  shared_paths:
    - storage/logs
    - public/uploads
hooks:
  post_deploy:
    - php artisan migrate --force
    - systemctl reload nginx
```

### Node.js Application
```yaml
app:
  name: nodeapp
  keep_releases: 3
server:
  host: node.example.com
  user: deploy
deploy:
  path: /opt/nodeapp
  shared_paths:
    - logs
    - uploads
  environment:
    NODE_ENV: production
hooks:
  post_deploy:
    - npm install --production
    - pm2 restart nodeapp
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `revlay init` | Initialize a new project |
| `revlay deploy` | Deploy a new release |
| `revlay rollback` | Rollback to previous release |
| `revlay releases` | List all releases |
| `revlay status` | Show deployment status |
| `revlay --help` | Show help information |

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
- `shared_paths`: Directories to share between releases
- `environment`: Environment variables

### Hooks Section
- `pre_deploy`: Commands to run before deployment
- `post_deploy`: Commands to run after deployment
- `pre_rollback`: Commands to run before rollback
- `post_rollback`: Commands to run after rollback

## Best Practices

1. **Use SSH keys** instead of passwords for authentication
2. **Test deployments** with `--dry-run` flag first
3. **Monitor releases** with `revlay status`
4. **Keep backups** of your configuration files
5. **Use shared paths** for persistent data
6. **Configure hooks** for application-specific tasks
7. **Set appropriate keep_releases** limit for your disk space

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

### Permission Issues
- Ensure deploy user has write access to deployment path
- Check shared directory permissions
- Verify hook commands can be executed by deploy user