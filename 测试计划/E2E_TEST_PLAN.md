# Revlay 端到端测试计划 (E2E Test Plan)

本文档旨在为 `revlay push` 命令重构后的新工作流提供一套完整的端到端测试方案。该方案基于 Docker 构建了一系列标准化的远程服务器环境，以系统性地验证各项功能。

## 1. 测试环境准备

测试环境的核心是位于 `docker-test` 目录下的 Dockerfile 和相关脚本。在开始测试前，请确保完成以下准备工作。

### 第 1 步：准备二进制文件

测试将同时需要一个“最新版”和“旧版”的 `revlay` 二进制文件。

1.  **编译最新版**: 在项目根目录执行 `go build -o revlay .` 生成最新版。
2.  **准备旧版**: 找到一个版本号低于最新版的 `revlay` 二进制文件，将其命名为 `revlay-old` 并放置在 `docker-test/` 目录下。

最终目录结构应如下：
```
.
├── docker-test/
│   ├── Dockerfile.base
│   ├── Dockerfile.new
│   ├── Dockerfile.old
│   ├── run_tests.sh
│   └── revlay-old      # 旧版二进制文件
└── revlay              # 最新版二进制文件
```

### 第 2 步：构建 Docker 测试镜像

我们为不同的测试场景定义了多个 Docker 镜像。

**重要提示**: `Dockerfile.base` 已被修改，增加了国内镜像源配置以解决 `apt` 网络问题。

请在项目根目录执行以下命令来构建所有测试镜像：

```bash
# 1. 构建包含 SSH 和基础工具的 base 镜像
docker build -t revlay-test-env:base -f docker-test/Dockerfile.base docker-test/

# 2. 构建安装了新版 revlay 的镜像
docker build -t revlay-test-env:new -f docker-test/Dockerfile.new .

# 3. 构建安装了旧版 revlay 的镜像
docker build -t revlay-test-env:old -f docker-test/Dockerfile.old docker-test/
```

### 第 3 步：启动测试环境

使用 `docker-test/run_tests.sh` 脚本可以一键启动指定的测试环境。该脚本会自动处理容器启停、SSH 密钥生成与注入等操作。

例如，启动一个纯净的、未安装 `revlay` 的远程环境：
```bash
./docker-test/run_tests.sh A
```

当测试完成后，在脚本运行的终端按下 `Ctrl+C` 即可自动销毁环境。

## 2. 测试用例

以下测试用例覆盖了从参数校验到复杂远程交互的全部流程。

---

### 第一部分：核心 `push` 流程与参数校验

#### Test Case 1.1: 必需参数校验
*   **目标**: 验证当 `push` 命令缺少必需的 `-p`, `-to`, `-app` 标志时，程序能优雅退出并提供清晰的错误提示。
*   **环境**: 无需 Docker 环境，直接在本地执行。
*   **执行命令**:
    ```bash
    # 缺少 -p
    ./revlay push -to revlay-user@localhost -app my-app

    # 缺少 -to
    ./revlay push -p . -app my-app

    # 缺少 -app
    ./revlay push -p . -to revlay-user@localhost
    ```
*   **预期结果**: 每条命令都执行失败，并打印出具体哪个标志缺失的错误信息。

#### Test Case 1.2: 本地环境预检 (手动测试)
*   **目标**: 验证 `revlay` 在执行前会检查本地是否存在 `ssh` 和 `rsync` 命令。
*   **环境**: 本地宿主机。
*   **执行操作**:
    1.  临时重命名本地的 `rsync` 命令：`sudo mv /usr/bin/rsync /usr/bin/rsync.bak` (路径可能因系统而异)。
    2.  执行任意 `revlay push` 命令。
    3.  恢复命令：`sudo mv /usr/bin/rsync.bak /usr/bin/rsync`。
*   **预期结果**: `revlay` 启动时检测到 `rsync` 不存在，立即终止并打印错误信息，提示用户需要安装 `rsync`。

---

### 第二部分：远程环境探测与交互

#### Test Case 2.1: 目标主机无 `revlay` (环境 A)
*   **目标**: 验证在纯净系统上，`push` 命令能触发远程自动安装 `revlay`。
*   **环境**: **A** (纯净系统)。 启动命令: `./docker-test/run_tests.sh A`
*   **执行命令**:
    ```bash
    ./revlay push -p . -to revlay-user@localhost -app my-app -i ./docker-test/revlay_test_key --ssh-port 2222
    ```
*   **预期结果**:
    1.  程序检测到远程 `revlay` 不存在。
    2.  显示 “正在远程安装 Revlay...” 等提示信息。
    3.  自动下载并安装与本地版本匹配的 `revlay` 到远程服务器。
    4.  安装成功后，流程继续，进入应用不存在的逻辑，并提问“是否立即进行初始化引导？”。

#### Test Case 2.2: 目标主机 `revlay` 版本过旧 (环境 D)
*   **目标**: 验证当远程 `revlay` 版本过旧且不兼容时，能触发自动更新。
*   **环境**: **D** (版本过旧)。 启动命令: `./docker-test/run_tests.sh D`
*   **执行命令**:
    ```bash
    ./revlay push -p . -to revlay-user@localhost -app my-app -i ./docker-test/revlay_test_key --ssh-port 2222
    ```
*   **预期结果**:
    1.  程序进行版本比对，发现远程版本不兼容。
    2.  触发远程 `revlay` 的自我更新流程。
    3.  更新成功后，程序打印“远程版本已更新，请重新执行命令”等提示信息后退出。
    4.  **验证**: 再次执行相同的 `push` 命令，程序不再提示更新，而是进入应用不存在的交互式引导流程。

#### Test Case 2.3: 远程应用不存在 (环境 B)
*   **目标**: 验证在远程 `revlay` 已安装但应用不存在时，能触发交互式初始化引导。
*   **环境**: **B** (空应用)。 启动命令: `./docker-test/run_tests.sh B`
*   **执行命令**:
    ```bash
    ./revlay push -p . -to revlay-user@localhost -app my-new-app -i ./docker-test/revlay_test_key --ssh-port 2222
    ```
*   **预期结果**:
    1.  版本检查通过。
    2.  应用检查失败，发现 `my-new-app` 不存在。
    3.  终端出现交互式提问：“应用 'my-new-app' 不存在。是否立即进行初始化引导？ (y/N)”。

#### Test Case 2.4: 远程应用已存在 (环境 C)
*   **目标**: 验证一次标准的、无障碍的 `push` 部署流程。
*   **环境**: **C** (应用存在)。 启动命令: `./docker-test/run_tests.sh C`
*   **执行命令**:
    ```bash
    ./revlay push -p . -to revlay-user@localhost -app my-app -i ./docker-test/revlay_test_key --ssh-port 2222
    ```
*   **预期结果**:
    1.  版本检查通过。
    2.  应用检查通过。
    3.  程序开始同步文件并执行远程部署。
    4.  所有步骤（探测、检查、同步、部署）都通过 `pterm.Spinner` 显示了清晰的状态。
    5.  最后打印部署成功的消息。

#### Test Case 2.5: 远程主机网络不通 (环境 E)
*   **目标**: 验证在无法建立 SSH 连接时，程序能返回清晰的网络错误。
*   **环境**: **E** (网络不通)。无需启动 Docker 容器。
*   **执行命令**:
    ```bash
    # 使用一个不存在的端口来模拟连接失败
    ./revlay push -p . -to revlay-user@localhost -app my-app -i ./docker-test/revlay_test_key --ssh-port 9999
    ```
*   **预期结果**: 命令执行失败，并打印出 SSH 连接被拒绝或连接超时的错误信息。

---

### 第三部分：功能标志与输出控制

#### Test Case 3.1: 日志详细模式 (`--verbose`)
*   **目标**: 验证 `--verbose` 标志能打印出底层的 SSH 和 rsync 命令输出。
*   **环境**: **C** (应用存在)。 启动命令: `./docker-test/run_tests.sh C`
*   **执行命令**:
    ```bash
    ./revlay push -p . -to revlay-user@localhost -app my-app -i ./docker-test/revlay_test_key --ssh-port 2222 --verbose
    ```
*   **预期结果**: 在标准的 Spinner 流程信息之外，还能看到详细的 `ssh` 和 `rsync` 执行日志流。

#### Test Case 3.2: 静默模式 (`--quiet`)
*   **目标**: 验证 `--quiet` 标志能抑制所有非必要的输出。
*   **环境**: **C** (应用存在)。 启动命令: `./docker-test/run_tests.sh C`
*   **执行命令**:
    ```bash
    ./revlay push -p . -to revlay-user@localhost -app my-app -i ./docker-test/revlay_test_key --ssh-port 2222 --quiet
    ```
*   **预期结果**: 屏幕上不显示任何 Spinner 或进度提示，只在部署成功后打印最终结果，或在失败时打印错误信息。

---

### 第四部分: 原子化更新 (手动验证)

#### Test Case 4.1: 验证原子化更新的安全性
*   **目标**: 通过观察，确认远程更新过程是原子的，具备备份和回滚能力。
*   **环境**: **D** (版本过旧)。 启动命令: `./docker-test/run_tests.sh D`
*   **执行操作**:
    1.  在一个终端中，执行 `push` 命令以触发更新。
    2.  在另一个终端，迅速连接到 Docker 容器内部：`docker exec -it revlay-remote-server bash`。
    3.  进入 `revlay` 所在目录：`cd /usr/local/bin`。
    4.  反复执行 `ls -l`，观察文件变化。
*   **预期结果**:
    1.  可以看到 `revlay` 被重命名为 `revlay.bak`。
    2.  一个新的临时文件（下载的新版本）出现。
    3.  新版本被验证后，`revlay.bak` 被删除，新版本被重命名为 `revlay`。整个替换过程是瞬间的（通过 `mv` 实现原子性）。 