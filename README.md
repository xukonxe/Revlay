<div align="center">

# Revlay

<p>
  <strong>一个现代、快速、无依赖的部署与服务器生命周期管理工具</strong>
</p>

<p>
  <img src="https://img.shields.io/github/v/release/xukonxe/Revlay?style=for-the-badge&logo=github" alt="Latest Release" />
  <img src="https://img.shields.io/github/license/xukonxe/Revlay?style=for-the-badge" alt="License" />
  <img src="https://img.shields.io/github/actions/workflow/status/xukonxe/Revlay/release.yml?style=for-the-badge&logo=githubactions" alt="Build Status" />
</p>

</div>

> **⚠️ Beta 阶段警告**
>
> 请注意，Revlay 目前正处于密集的 **Beta 开发阶段**。这意味着：
> *   API 和命令行接口可能会发生**不兼容的变更**。
> *   可能会存在一些未知的 Bug 或稳定性问题。
>
> 我们非常欢迎您试用、提供反馈和参与贡献，但**请勿在关键的生产环境中使用**。

## ✨ What is Revlay?

你是否曾羡慕 Heroku 或 Vercel 那样优雅的 `git push` 一键部署体验，但又想完全掌控自己的服务器？Revlay 就是为此而生。

**Revlay** 是一款开源的部署工具，它将现代化的部署理念带到您的私有服务器上。它借鉴了 Capistrano 的原子化部署思想，通过创建带有时间戳的发布版本和切换符号链接来实现零停机、可快速回滚的部署。

最重要的是，一旦 Revlay 被安装在您的服务器上，它**不依赖于任何外部服务或运行时**（如 Docker, Git），为您提供了一个轻量、高速且易于维护的部署解决方案。

## 🚀 主要特性

*   **原子化部署 (Atomic Deploys):** 每一次部署都是一个独立的版本。通过切换 `current` 符号链接，发布和回滚操作几乎是瞬时完成的，并且风险极低。
*   **一键远程部署:** 只需在本地运行 `revlay push`，即可将代码打包、上传到服务器并自动触发部署流程，全程自动化。
*   **零停机更新:** 内置的 TCP 代理支持 `zero_downtime` 部署模式，确保您的服务在更新期间持续可用。
*   **版本管理:** 轻松列出所有历史版本 (`releases`)，并可一键回滚 (`rollback`) 到任何一个旧版本。
*   **服务生命周期管理:** 统一管理服务器上的多个应用，包括启动 (`start`)、停止 (`stop`) 和查看状态 (`ps`)。
*   **内置自我更新:** 只需运行 `revlay update`，即可将 Revlay 自身更新到最新版本，无需重复安装。
*   **美化的交互界面:** 使用 `gum` 和 `pterm` 提供了现代化的、信息丰富的命令行交互体验。

## ⚡️ 快速安装

我们提供了一个智能安装脚本，它会自动检测您的操作系统和架构，并为您安装最新版本的 Revlay。

在您的 macOS 或 Linux 终端中运行以下命令即可：

```bash
curl -fsSL https://raw.githubusercontent.com/xukonxe/Revlay/main/scripts/install.sh | bash
```
> **注意:** 脚本可能会因为需要将 `revlay` 安装到 `/usr/local/bin` 而提示您输入密码。

## 🏁 快速上手

在 **5分钟内** 完成您的第一次部署！

1.  **安装 Revlay**
    按照上面的方法，在您的**本地开发机**和**目标服务器**上都安装好 Revlay。

2.  **在本地初始化项目**
    ```bash
    # 这会在当前目录下创建一个名为 my-awesome-app 的文件夹
    # 和一个基础的 revlay.yml 配置文件
    revlay init my-awesome-app
    cd my-awesome-app
    ```

3.  **在服务器上注册服务**
    登录到您的服务器，并告诉 Revlay 您的应用将部署在哪里。
    ```bash
    # 'myapp' 是你给这个服务起的名字，可以任意
    # '/var/www/myapp' 是你希望部署应用的服务器绝对路径
    revlay service add myapp /var/www/myapp
    ```

4.  **从本地推送并部署**
    回到您的本地开发机，将您的应用代码（例如 `dist` 目录）推送到服务器。
    ```bash
    # 确保您的开发机可以通过 SSH 免密登录到服务器
    # 将 'user@your-server.com' 替换成您的服务器地址
    # --to myapp 告诉 revlay 这是针对刚才注册的那个服务
    revlay push ./dist to user@your-server.com --to myapp
    ```
    恭喜！您的应用已经成功部署。

## 🛠️ 命令概览

| 命令 | 别名 | 描述 |
| :--- | :--- | :--- |
| `deploy` | | （在服务器上）从一个目录部署新版本 |
| `push` | | 将本地目录推送到服务器并触发部署 |
| `rollback` | | 回滚到上一个或指定的版本 |
| `releases` | | 列出所有已部署的版本 |
| `status` | | 显示当前部署的状态 |
| `service` | | 管理服务 (add, remove, list) |
| `ps` | `service list` | 列出所有已注册的服务及其状态 |
| `start` | `service start` | 启动一个已部署的服务 |
| `stop` | `service stop` | 停止一个正在运行的服务 |
| `update` | | 将 Revlay 程序自身更新到最新版 |
| `init` | | 初始化一个新的 Revlay 项目 |

## ❤️ 参与贡献

我们热烈欢迎各种形式的贡献！无论是提交 Bug、提出新功能建议还是直接贡献代码。

*   **报告问题:** 请通过 [GitHub Issues](https://github.com/xukonxe/Revlay/issues) 来提交。
*   **贡献代码:** 请 Fork 本仓库，创建您的功能分支，然后提交 Pull Request。

## 📄 授权协议

本项目基于 [MIT License](LICENSE) 授权。