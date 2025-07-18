# Revlay 功能测试计划：基于 Docker 的端到端验证

## 1. 核心思想

为了系统性地测试 `revlay push` 在不同远程服务器环境下的行为，我们采用**多镜像策略**。为每种关键的服务器状态构建一个独立的 Docker 镜像，从而实现测试环境的标准化和一键切换。

这种方法将环境的“构建”和“运行”彻底分离，使测试更加清晰、可复现。

**我们将定义的五种核心测试环境：**
*   **环境 A (纯净系统)**: 一个干净的 Linux 系统，未安装 `revlay`。
*   **环境 B (空应用)**: 安装了兼容的新版 `revlay`，但未配置任何应用。
*   **环境 C (应用存在)**: 安装了新版 `revlay`，并已配置了一个名为 `my-app` 的应用。
*   **环境 D (版本过旧)**: 安装了一个不兼容的旧版 `revlay`。
*   **环境 E (网络不通)**: 这是一个运行时场景，而非镜像。通过连接错误的地址或关闭容器来模拟。

---

## 2. 多 Dockerfile 环境构建

在项目根目录下创建一个 `docker-test` 目录来存放所有测试相关的文件。

```
.
├── docker-test/
│   ├── Dockerfile.base         # A: 基础系统镜像
│   ├── Dockerfile.new          # B: 包含新版 revlay
│   ├── Dockerfile.old          # D: 包含旧版 revlay
│   ├── run_tests.sh            # 自动化测试脚本
│   └── revlay-old              # 【需手动准备】一个旧版二进制文件
└── revlay                      # 【需手动准备】新版二进制文件
```

### 第 1 步：创建 `Dockerfile.base` (环境 A)

这个文件定义了包含 SSH 服务的纯净 Linux 系统。

**`docker-test/Dockerfile.base`**:
```dockerfile
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-server rsync curl wget sudo && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash revlay-user && \
    echo "revlay-user:revlay-password" | chpasswd && \
    adduser revlay-user sudo

RUN mkdir /var/run/sshd
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
```

### 第 2 步：创建 `Dockerfile.new` (环境 B 的基础)

这个文件继承自基础镜像，并安装**新版**的 `revlay`。

**`docker-test/Dockerfile.new`**:
```dockerfile
FROM revlay-test-env:base
# 将项目根目录的新版 revlay 复制到镜像中
COPY ./revlay /usr/local/bin/revlay
RUN chmod +x /usr/local/bin/revlay
```

### 第 3 步：创建 `Dockerfile.old` (环境 D)

这个文件同样继承自基础镜像，但安装的是一个**旧版** `revlay`。

**`docker-test/Dockerfile.old`**:
```dockerfile
FROM revlay-test-env:base
# 将 docker-test 目录下的旧版 revlay 复制到镜像中
COPY ./revlay-old /usr/local/bin/revlay
RUN chmod +x /usr/local/bin/revlay
```

### 第 4 步：构建所有镜像

```bash
# 进入测试目录
cd docker-test

# 1. 构建基础镜像
docker build -t revlay-test-env:base -f Dockerfile.base .

# 2. 构建新版 revlay 镜像 (需要上一级目录的 revlay 文件)
docker build -t revlay-test-env:new -f Dockerfile.new ../

# 3. 构建旧版 revlay 镜像 (需要此目录的 revlay-old 文件)
docker build -t revlay-test-env:old -f Dockerfile.old .

# 返回项目根目录
cd ..
```

---

## 3. 自动化测试脚本 (`run_tests.sh`)

这个脚本将成为我们测试的入口。它可以根据参数启动并配置指定的环境。

**`docker-test/run_tests.sh`**:
```bash
#!/bin/bash
set -e # 如果命令失败则退出

# --- 配置 ---
CONTAINER_NAME="revlay-remote-server"
SSH_USER="revlay-user"
SSH_PORT=2222
SSH_KEY_PATH="./revlay_test_key"
ENV_TYPE=$1

# --- 清理旧容器 ---
docker rm -f ${CONTAINER_NAME} > /dev/null 2>&1 || true

# --- 选择要启动的镜像 ---
case $ENV_TYPE in
  "A")
    IMAGE_NAME="revlay-test-env:base"
    echo ">> 启动环境 A (纯净系统)..."
    ;;
  "B")
    IMAGE_NAME="revlay-test-env:new"
    echo ">> 启动环境 B (空应用)..."
    ;;
  "C")
    IMAGE_NAME="revlay-test-env:new"
    echo ">> 启动环境 C (应用存在)..."
    ;;
  "D")
    IMAGE_NAME="revlay-test-env:old"
    echo ">> 启动环境 D (版本过旧)..."
    ;;
  *)
    echo "错误: 请提供环境类型 (A, B, C, D)"
    exit 1
    ;;
esac

# --- 通用启动流程 ---
# 生成测试密钥（如果不存在）
[ -f "$SSH_KEY_PATH" ] || ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_PATH" -N ""
SSH_PUB_KEY_CONTENT=$(cat "${SSH_KEY_PATH}.pub")

docker run -d --rm -p ${SSH_PORT}:22 --name ${CONTAINER_NAME} ${IMAGE_NAME}
sleep 3 # 等待 sshd 启动

# 注入公钥
docker exec ${CONTAINER_NAME} bash -c "mkdir -p /home/${SSH_USER}/.ssh && \
  echo '${SSH_PUB_KEY_CONTENT}' > /home/${SSH_USER}/.ssh/authorized_keys && \
  chown -R ${SSH_USER}:${SSH_USER} /home/${SSH_USER}/.ssh && \
  chmod 700 /home/${SSH_USER}/.ssh && chmod 600 /home/${SSH_USER}/.ssh/authorized_keys"

# --- 特定环境配置 ---
if [ "$ENV_TYPE" == "C" ]; then
  echo ">> 为环境 C 配置应用..."
  docker exec --user ${SSH_USER} ${CONTAINER_NAME} revlay service add --name my-app --from "http://example.com"
fi

echo ">> 测试环境 '${ENV_TYPE}' 准备就绪!"
echo "   - 目标主机: ${SSH_USER}@localhost"
echo "   - SSH 端口: ${SSH_PORT}"
echo "   - SSH 私钥: ${SSH_KEY_PATH}"
echo ">> 按 Ctrl+C 销毁环境。"

# 保持脚本运行以便手动测试，Ctrl+C 会触发 trap
trap "echo; echo '>> 销毁容器...'; docker rm -f ${CONTAINER_NAME} > /dev/null" EXIT
sleep infinity
```
**授权脚本**: `chmod +x docker-test/run_tests.sh`

---

## 4. 执行测试

现在，你可以按需启动任何一个测试环境。

1.  **测试环境 A (纯净系统)**:
    `./docker-test/run_tests.sh A`
    *   在另一个终端执行 `revlay push`，预期会收到 “远程 revlay 未安装” 的相关提示。

2.  **测试环境 B (空应用)**:
    `./docker-test/run_tests.sh B`
    *   执行 `revlay push -app non-exist-app`，预期会收到 “应用不存在” 并触发交互式引导。

3.  **测试环境 C (应用存在)**:
    `./docker-test/run_tests.sh C`
    *   执行 `revlay push -app my-app`，预期应用检查通过，流程继续。

4.  **测试环境 D (版本过旧)**:
    `./docker-test/run_tests.sh D`
    *   执行 `revlay push`，预期版本检查逻辑会发现不兼容。

5.  **测试环境 E (网络不通)**:
    *   这个场景无需启动脚本。直接在 `revlay push` 中使用一个错误的端口或地址即可，例如 `--ssh-port 9999`。预期会收到连接超时或拒绝连接的错误。

当测试完成后，回到运行脚本的终端，按下 `Ctrl+C`，脚本会自动清理并删除正在运行的容器。 