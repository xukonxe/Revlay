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

## 4. 详细测试用例

现在，你可以按需启动任何一个测试环境来执行具体的端到端测试。每个测试用例都旨在验证 `25.7.18.md` 中记录的特定功能点。

**启动环境**: 在项目根目录执行 `./docker-test/run_tests.sh <环境代码>` (例如 `./docker-test/run_tests.sh A`)
**执行测试**: 在另一个终端中，进入项目根目录执行 `revlay push ...` 命令。
**清理环境**: 完成测试后，在运行脚本的终端按下 `Ctrl+C`。

### 场景一：远程服务器为纯净系统 (环境 A)

**目标**: 验证远程首次安装、版本握手、应用初始化引导的完整流程。

| 测试命令 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| `revlay push -to revlay-user@localhost --ssh-port 2222 -i ./revlay_test_key -p . -app my-app` | 1. 探测到远程未安装 `revlay`。<br/>2. 提示用户是否安装，选择 `y`。<br/>3. 自动从 GitHub Release 下载并**原子化安装**最新版 `revlay`。<br/>4. 重新探测，版本握手成功。<br/>5. 探测到应用 `my-app` 不存在。<br/>6. 提示用户是否初始化，选择 `y`。<br/>7. `push` 流程继续，最终部署成功。 | - 远程 `revlay` 探测<br/>- 交互式远程安装<br/>- 原子化安装流程<br/>- 应用不存在时的交互式引导 |
| *在上述流程中，对安装提示选择 `n`* | 命令终止，并提示用户需要手动安装。 | - 拒绝安装时的优雅退出 |
| *在上述流程中，对应用初始化提示选择 `n`* | 命令终止，并提示应用不存在。 | - 拒绝初始化时的优雅退出 |

### 场景二：远程已安装 `revlay` 但无应用 (环境 B)

**目标**: 验证已安装 `revlay` 时的应用探测和初始化流程。

| 测试命令 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| `revlay push -to revlay-user@localhost --ssh-port 2222 -i ./revlay_test_key -p . -app non-exist-app` | 1. 版本握手成功。<br/>2. 探测到应用 `non-exist-app` 不存在。<br/>3. 触发交互式初始化引导。<br/>4. `push` 流程继续。 | - `service list --output=json`<br/>- 应用不存在时的交互式引导 |

### 场景三：远程已就绪（`revlay` 和应用均存在） (环境 C)

**目标**: 验证核心 `push` 流程（快乐路径）以及 UI 标志。

| 测试命令 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| `revlay push -to revlay-user@localhost --ssh-port 2222 -i ./revlay_test_key -p . -app my-app` | 1. 版本握手成功。<br/>2. 应用检查通过。<br/>3. 文件通过 `rsync` 同步。<br/>4. 远程部署成功。<br/>5. 全程有 `pterm` Spinner 动画。 | - 核心部署 Happy Path<br/>- `pterm` 可视化反馈 |
| `... --verbose` (在上述命令后追加) | 除了正常流程，还输出 `ssh` 和 `rsync` 的详细执行日志。 | - `--verbose` 标志 |
| `... --quiet` (在上述命令后追加) | 不显示任何 Spinner，只在出错时打印错误信息。 | - `--quiet` 标志 |

### 场景四：远程 `revlay` 版本过旧 (环境 D)

**目标**: 验证版本不兼容时的自动更新机制和 `ErrRemoteUpdated` 错误处理。

| 测试命令 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| `revlay push -to revlay-user@localhost --ssh-port 2222 -i ./revlay_test_key -p . -app my-app` | 1. 版本握手失败，检测到远程版本过旧。<br/>2. 提示用户需要升级，选择 `y`。<br/>3. 触发**原子化更新**流程。<br/>4. 更新成功后，程序退出，并提示用户 “远程环境已更新，请重新执行 push 命令”。 | - 版本兼容性策略<br/>- 原子化更新流程<br/>- `core.ErrRemoteUpdated` 的 CLI 处理 |

### 场景五：网络或认证失败 (环境 E)

**目标**: 验证 SSH 连接异常的错误处理。

| 测试命令 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| `revlay push -to revlay-user@localhost --ssh-port 9999 ...` | 命令失败，打印清晰的 SSH 连接超时或拒绝连接的错误。 | - 优雅的 SSH 错误处理 |
| `revlay push -to revlay-user@localhost --ssh-key /path/to/wrong/key ...` | 命令失败，打印清晰的 SSH 认证失败错误。 | - 优雅的 SSH 错误处理 |

### 场景六：必要参数校验

**目标**: 验证 `push` 命令自身的参数校验。此测试无需启动 Docker 环境。

| 测试命令 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| `revlay push` | 命令失败，提示缺少 `-to`, `-p`, `-app` 等必要参数。 | - `pflag` 参数校验 |

### 场景七：本地环境预检 (手动测试)

**目标**: 验证 `ssh` 和 `rsync` 的本地依赖检查。

| 测试步骤 | 预期行为 | 验证功能点 |
| :--- | :--- | :--- |
| 1. 找一台未安装 `rsync` 的机器。<br/>2. 在该机器上执行任意 `revlay push` 命令。 | 程序启动时立即终止，并打印清晰的错误信息，提示 `rsync` 未安装并给出安装指引。 | - 本地环境预检 |

---

## 5. 对测试框架的补充建议

当前基于 Docker 的测试框架非常出色。为了支持更复杂的测试场景，例如**测试原子化更新的回滚能力**，可以考虑增加一个“损坏的”更新环境。

**建议的环境 F (模拟更新失败)**:

1.  **准备一个损坏的 `revlay` 二进制文件**: 可以是一个空文件或一个无法执行的脚本，命名为 `revlay-broken`。
2.  **修改 `run_tests.sh`**:
    *   启动环境 D (旧版本)。
    *   在容器内，通过 `docker exec` 修改 `/etc/hosts` 或使用其他网络工具，将 `github.com` 的流量重定向到一个本地 HTTP 服务器。
    *   这个本地服务器托管我们准备好的 `revlay-broken` 文件，并使其下载链接与 `revlay` 的真实下载链接匹配。
3.  **执行测试**:
    *   运行 `revlay push`。
    *   **预期**: 更新器下载了损坏的文件 -> `chmod +x` 和 `--version` 验证失败 -> 触发回滚 -> `push` 失败并报告更新错误 -> SSH 登录容器检查，确认旧版 `revlay` 仍然存在。

这个场景的设置较为复杂，可以作为后续的进阶测试任务。 