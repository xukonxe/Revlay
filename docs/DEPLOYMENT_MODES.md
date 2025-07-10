# Revlay 部署模式详解

## 概述

Revlay 支持两种不同的部署模式，用于处理不同的应用场景和部署需求：

1. **零停机部署** (Zero Downtime Deployment) - 默认模式
2. **短停机部署** (Short Downtime Deployment) - 传统模式

## 部署模式配置

在 `revlay.yml` 配置文件中设置部署模式：

```yaml
deploy:
  mode: zero_downtime  # 或 short_downtime
```

## 零停机部署模式 (Zero Downtime)

### 工作原理

零停机部署使用蓝绿部署策略：

1. **端口管理**: 使用主端口和备用端口交替运行服务
2. **健康检查**: 新服务启动后进行健康检查确保可用
3. **流量切换**: 通过负载均衡器或反向代理切换流量
4. **优雅关闭**: 旧服务在流量切换后优雅关闭

### 配置示例

```yaml
app:
  name: myapp
  keep_releases: 5

server:
  host: server.example.com
  user: deploy
  port: 22

deploy:
  path: /opt/myapp
  mode: zero_downtime
  shared_paths:
    - storage/logs
    - storage/uploads

service:
  command: "cd ${RELEASE_PATH} && PORT=${PORT} node server.js"
  port: 8080
  alt_port: 8081
  health_check: "/health"
  restart_delay: 5
  graceful_timeout: 30

hooks:
  post_deploy:
    - "nginx -s reload"  # 重新加载 nginx 配置
```

### 部署流程

1. 创建新版本目录
2. 在备用端口启动新服务
3. 健康检查验证新服务
4. 更新符号链接指向新版本
5. 切换负载均衡器到新端口
6. 优雅关闭旧服务
7. 清理旧版本

### 适用场景

- 无文件锁定的应用
- 无数据库锁定的应用
- 支持多实例运行的应用
- 有负载均衡器的环境

## 短停机部署模式 (Short Downtime)

### 工作原理

短停机部署使用传统的停止-更新-启动策略：

1. **服务停止**: 优雅停止当前服务
2. **版本切换**: 更新符号链接到新版本
3. **服务启动**: 启动新版本服务
4. **验证**: 确认服务正常运行

### 配置示例

```yaml
app:
  name: myapp
  keep_releases: 5

server:
  host: server.example.com
  user: deploy
  port: 22

deploy:
  path: /opt/myapp
  mode: short_downtime
  shared_paths:
    - storage/logs
    - storage/uploads
    - data/database.db  # 共享数据库文件

service:
  command: "systemctl restart myapp"
  port: 8080
  graceful_timeout: 30

hooks:
  pre_deploy:
    - "systemctl stop myapp"
  post_deploy:
    - "systemctl start myapp"
    - "systemctl reload nginx"
```

### 部署流程

1. 运行预部署钩子（停止服务）
2. 创建新版本目录
3. 更新符号链接到新版本
4. 运行后部署钩子（启动服务）
5. 清理旧版本

### 适用场景

- 有文件锁定的应用
- 有数据库锁定的应用
- 启动时需要全局加载配置的应用
- 单实例应用
- 不支持多端口的应用

## 端口冲突解决方案

### 问题分析

在零停机部署中，新旧版本可能会产生端口冲突。Revlay 通过以下方式解决：

1. **端口轮换**: 使用主端口和备用端口交替
2. **端口检测**: 自动检测当前使用的端口
3. **健康检查**: 确保新服务在备用端口正常运行
4. **流量切换**: 通过配置文件或负载均衡器切换

### 负载均衡器配置

#### Nginx 示例

```nginx
upstream myapp {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name example.com;
    
    location / {
        proxy_pass http://myapp;
    }
}
```

使用脚本自动切换端口：

```bash
#!/bin/bash
# /opt/myapp/scripts/switch_port.sh

CURRENT_PORT=$(cat /opt/myapp/current_port)
if [ "$CURRENT_PORT" = "8080" ]; then
    NEW_PORT=8081
else
    NEW_PORT=8080
fi

# 更新 nginx 配置
sed -i "s/server 127.0.0.1:[0-9]*/server 127.0.0.1:$NEW_PORT/" /etc/nginx/sites-available/myapp
nginx -s reload

# 更新端口文件
echo $NEW_PORT > /opt/myapp/current_port
```

#### HAProxy 示例

```
backend myapp
    server app1 127.0.0.1:8080 check
    server app2 127.0.0.1:8081 check backup
```

## 数据库和文件锁定处理

### 零停机部署的限制

零停机部署在以下情况下可能遇到问题：

1. **数据库锁定**: 新旧版本同时访问数据库可能导致锁冲突
2. **文件锁定**: 共享文件可能被多个进程锁定
3. **状态共享**: 内存中的状态在新旧版本间无法共享

### 解决方案

#### 1. 数据库迁移策略

```yaml
hooks:
  pre_deploy:
    - "php artisan migrate --force"  # 向前兼容的迁移
  post_deploy:
    - "sleep 30"  # 等待旧实例处理完请求
    - "php artisan migrate:cleanup"  # 清理旧的迁移数据
```

#### 2. 文件锁定避免

```yaml
shared_paths:
  - storage/logs
  - storage/uploads
  - storage/cache  # 避免缓存冲突
  - storage/sessions  # 共享会话存储

service:
  command: "cd ${RELEASE_PATH} && PORT=${PORT} CACHE_PREFIX=v${PORT} node server.js"
```

#### 3. 使用外部存储

```yaml
deploy:
  environment:
    REDIS_URL: "redis://localhost:6379"
    DB_HOST: "localhost"
    SESSION_DRIVER: "redis"
    CACHE_DRIVER: "redis"
```

## 最佳实践

### 选择部署模式的建议

| 应用类型 | 推荐模式 | 原因 |
|---------|---------|------|
| Web API (无状态) | 零停机 | 易于多实例运行 |
| 微服务 | 零停机 | 独立部署，无状态 |
| 传统单体应用 | 短停机 | 可能有状态依赖 |
| 数据库密集型应用 | 短停机 | 避免连接冲突 |
| 文件处理应用 | 短停机 | 避免文件锁定 |

### 配置检查清单

#### 零停机部署

- [ ] 应用支持多实例运行
- [ ] 配置了健康检查端点
- [ ] 设置了主端口和备用端口
- [ ] 配置了负载均衡器
- [ ] 使用外部数据存储（Redis/数据库）
- [ ] 避免文件锁定

#### 短停机部署

- [ ] 配置了服务管理命令
- [ ] 设置了优雅关闭超时
- [ ] 配置了预/后部署钩子
- [ ] 处理了共享文件路径
- [ ] 考虑了数据备份

## 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   netstat -tlnp | grep :8080
   ```

2. **健康检查失败**
   ```bash
   curl -f http://localhost:8080/health
   ```

3. **服务启动失败**
   ```bash
   systemctl status myapp
   journalctl -u myapp -f
   ```

### 调试命令

```bash
# 查看当前部署状态
revlay status

# 预览部署计划
revlay deploy --dry-run

# 查看所有版本
revlay releases

# 测试 SSH 连接
ssh user@host "echo 'connection test'"
```

## 命令参考

### 语言支持

```bash
# 使用中文（默认）
revlay status

# 使用英文
revlay --lang=en status

# 设置环境变量
export REVLAY_LANG=en
revlay status
```

### Dry-Run 功能

Dry-run 是预览模式，显示将要执行的操作但不实际执行：

```bash
# 预览部署
revlay deploy --dry-run

# 显示详细的部署计划
# - 目录结构
# - 共享路径链接
# - 钩子命令
# - 部署模式配置
# - 服务端口设置
```

这个功能帮助用户：
- 验证配置正确性
- 了解部署流程
- 避免意外操作
- 调试部署问题

## 总结

Revlay 通过提供两种部署模式，满足了不同应用的部署需求：

- **零停机部署**: 适用于现代云原生应用，提供真正的零停机体验
- **短停机部署**: 适用于传统应用，提供可靠的部署方案

通过合理的配置和最佳实践，可以有效解决端口冲突、数据库锁定和文件锁定等问题。