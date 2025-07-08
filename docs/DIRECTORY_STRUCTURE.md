# 项目目录结构详解

这是 Revlay 项目的目录结构、各部分职责以及推荐的开发流程。

---

## 顶层目录

### `cmd/` - 程序入口

这是您所有二进制文件的"启动器"。目录下的每个子目录都对应一个可执行文件。

-   **`cmd/revlay/main.go`**: 这是 Revlay CLI 的主入口。它的职责非常简单：解析命令行参数，然后调用 `internal/cli` 包中的代码来执行真正的逻辑。
-   **`cmd/revlay-agent/main.go`**: 这是 `revlay-agent` 的主入口。它负责启动Agent服务，监听来自客户端的指令。

### `internal/` - 项目的心脏 (最重要的部分)

这是您项目的所有私有代码，外部应用无法导入。这强制了良好的模块化，是项目的核心所在。

-   **`internal/cli/`**: CLI的实现中心。推荐使用 [Cobra](https://github.com/spf13/cobra) 库来构建命令行。这里会定义 `deploy`, `rollback` 等所有命令，并编排它们的执行流程。
-   **`internal/deployment/`**: 部署流程的核心逻辑。例如 `Deployer` 结构体，包含 `Upload()`, `SwitchSymlink()`, `Cleanup()` 等方法。这部分代码是纯粹的业务逻辑，不关心它是被CLI还是被Agent调用。
-   **`internal/config/`**: 配置管理。负责加载、解析、验证 `revlay.yml` 文件。
-   **`internal/ssh/`**: SSH功能封装。将Go的SSH库封装成更易用的高层API，例如 `Connect()`, `RunCommand()`, `UploadFile()`。
-   **`internal/agent/`**: Agent的实现中心。负责解析来自客户端的请求，执行本地操作（如获取CPU信息），并返回结构化数据。
-   **`internal/client/`**: Agent客户端库。封装了通过SSH调用Agent API的逻辑。这样，您的CLI和UI客户端都可以使用这个包来与Agent通信，而不用重复编写底层代码。
-   **`internal/util/`**: 存放一些通用的辅助函数，比如日志记录器、文件操作等。

### `api/` - 契约与接口

当您的 Client 和 Agent 需要通信时，它们需要一个共同的"语言"，即API定义。把这个放在顶层，表示它是一个需要被多方（CLI, Agent, UI）共同遵守的契约。初期可以是简单的JSON，后期可以升级为更高效的 [Protobuf](https://protobuf.dev/)。

### `ui/` - 图形界面

这是一个独立的目录，专门存放所有UI相关的代码。这使得UI的开发可以与后端逻辑完全分离。

-   **`ui/desktop/`**: 桌面客户端的源代码。
-   **`ui/mobile/`**: 移动客户端的源代码。

### `pkg/` - 公共库 (可选)

如果您希望 Revlay 的某些功能可以被其他外部Go项目作为库来使用（例如，一个通用的SSH操作库），您可以把它放在这里。初期建议将所有代码都放在 `internal/` 中，直到您有明确的需求要将某些部分开源为独立的库。

### `.github/` - 社区与自动化

-   **`workflows/`**: 存放GitHub Actions的配置文件，用于自动化测试、构建和发布。
-   **`ISSUE_TEMPLATE/`**: 定义清晰的Issue模板，方便用户提交Bug报告和功能请求。

---

## 开发流程建议

1.  **从 `internal/` 开始**: 首先在 `internal/` 中实现核心逻辑，比如 `deployment` 和 `ssh` 包。编写单元测试来保证它们的质量。
2.  **构建CLI**: 在 `cmd/revlay/` 和 `internal/cli/` 中，将您写好的核心逻辑组装成一个可用的命令行工具。
3.  **演进到Agent**: 当您需要更复杂的操作时，开始开发 `internal/agent/` 和 `cmd/revlay-agent/`。同时开发 `internal/client/` 来调用它。
4.  **开发UI**: 最后，在 `ui/` 目录下开始您的UI客户端开发，UI客户端将通过 `internal/client/` 包与Agent通信。 