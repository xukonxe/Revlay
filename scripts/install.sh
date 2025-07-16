#!/bin/sh
# Revlay - 智能安装脚本 v2.1
#
# 特性:
# - 使用 gum 美化输出 (强制依赖)
# - 自动检测操作系统和架构
# - 支持安装指定版本
# - 支持自定义安装目录
# - 安装后检查 PATH

set -e

# --- 依赖检查 ---
if ! command -v curl >/dev/null; then
  echo "错误: 'curl' 未安装，无法继续。" >&2
  exit 1
fi
if ! command -v gum >/dev/null; then
  # gum 是一个硬性依赖，如果不存在则报错退出
  # 我们在这里尝试用 gum 来美化错误信息，如果 gum 不存在，这行会静默失败，但下面的 echo 会正常工作
  gum style --border normal --padding "1 2" --border-foreground 99 "错误: 依赖项 'gum' 未安装。" >/dev/null 2>&1 || true
  echo "错误: 'gum' 是此脚本的必需依赖项。"
  echo "请先通过 Homebrew 安装: brew install gum"
  exit 1
fi

# --- 变量定义 ---
OWNER="xukonxe"
REPO="Revlay"
INSTALL_DIR_DEFAULT="/usr/local/bin"
INSTALL_DIR="${INSTALL_DIR:-$INSTALL_DIR_DEFAULT}"
TARGET_BINARY_NAME="revlay"

# --- 脚本主逻辑 ---
main() {
  gum style --padding "1 2" --border double --border-foreground 212 \
    "欢迎使用 Revlay 智能安装脚本"

  # 确定要安装的版本
  if [ -n "$1" ]; then
    VERSION="$1"
    echo "准备安装指定版本: $(gum style --foreground 212 "$VERSION")"
  else
    echo "正在获取最新版本号..."
    VERSION=$(curl -s "https://api.github.com/repos/${OWNER}/${REPO}/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')
    if [ -z "$VERSION" ]; then
      echo "错误: 无法获取最新的 release 版本。" >&2; exit 1
    fi
    echo "最新版本为: $(gum style --foreground 212 "$VERSION")"
  fi

  # 检测系统信息
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)
  case "$ARCH" in
    "x86_64") ARCH="amd64" ;;
    "aarch64") ARCH="arm64" ;;
    "arm64") ARCH="arm64" ;;
  esac
  echo "检测到您的系统为: $(gum style --foreground 212 "$OS/$ARCH")"

  # 构建下载链接
  DOWNLOAD_URL="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}/revlay_${VERSION#v}_${OS}_${ARCH}.tar.gz"

  # 下载和解压
  TEMP_DIR=$(mktemp -d)
  gum spin --spinner dot --title "正在从 GitHub 下载资源..." -- \
    curl -L --progress-bar "${DOWNLOAD_URL}" | tar -xzf - -C "${TEMP_DIR}"

  # 安装
  INSTALL_PATH="${INSTALL_DIR}/${TARGET_BINARY_NAME}"
  echo "准备将 ${TARGET_BINARY_NAME} 安装到 $(gum style --foreground 212 "$INSTALL_PATH")..."

  # 如果目标目录不存在，则创建
  if [ ! -d "$INSTALL_DIR" ]; then
    gum spin --spinner dot --title "创建安装目录 ${INSTALL_DIR} (可能需要密码)..." -- \
      sudo mkdir -p "$INSTALL_DIR"
  fi

  gum spin --spinner dot --title "移动二进制文件并设置权限 (可能需要密码)..." -- \
    sudo mv "${TEMP_DIR}/${TARGET_BINARY_NAME}" "$INSTALL_PATH" && sudo chmod +x "$INSTALL_PATH"

  # 清理
  rm -rf "${TEMP_DIR}"

  # 检查 PATH
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*)
      # 在 PATH 中，一切正常
      ;;
    *)
      gum style --border normal --padding "1 2" --border-foreground 220 \
        "安装完成，但有一个小提示！" \
        "安装目录 '${INSTALL_DIR}' 不在您的 PATH 环境变量中。" \
        "请将以下命令添加到您的 shell 配置文件中 (如 ~/.zshrc 或 ~/.bash_profile):" \
        "" \
        "$(gum style --bold "export PATH=\"\$PATH:${INSTALL_DIR}\"")"
      ;;
  esac

  echo
  gum style --bold --foreground 212 "🎉 Revlay ${VERSION} 已成功安装！"
  echo "请运行 '$(gum style --foreground 212 "${TARGET_BINARY_NAME} --help")' 来开始使用。"
}

# 调用主函数，并将脚本参数传递进去
main "$@"