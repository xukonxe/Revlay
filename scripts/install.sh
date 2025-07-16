#!/bin/sh
# Revlay - 智能安装脚本
#
# 这个脚本会自动检测您的操作系统和架构，
# 然后从 GitHub Releases 下载最新版本的 Revlay 并安装到 /usr/local/bin。

set -e

# 检测操作系统和架构
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

# 通过 GitHub API 获取最新版本号
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/xukonxe/Revlay/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')

if [ -z "$LATEST_RELEASE" ]; then
    echo "错误: 无法获取最新的 release 版本。"
    exit 1
fi

# 构建下载链接
DOWNLOAD_URL="https://github.com/xukonxe/Revlay/releases/download/${LATEST_RELEASE}/revlay_${LATEST_RELEASE#v}_${OS}_${ARCH}.tar.gz"

echo "正在下载 Revlay ${LATEST_RELEASE} for ${OS}/${ARCH}..."

# 下载并解压到临时目录
TEMP_DIR=$(mktemp -d)
curl -L --progress-bar "${DOWNLOAD_URL}" | tar -xzf - -C "${TEMP_DIR}"

echo "正在安装到 /usr/local/bin..."

# 将二进制文件移动到 /usr/local/bin (可能需要 sudo)
# 并确保它有可执行权限
if [ -w "/usr/local/bin" ]; then
    mv "${TEMP_DIR}/revlay" "/usr/local/bin/revlay"
    chmod +x "/usr/local/bin/revlay"
else
    echo "提示: /usr/local/bin 不可写，尝试使用 sudo..."
    sudo mv "${TEMP_DIR}/revlay" "/usr/local/bin/revlay"
    sudo chmod +x "/usr/local/bin/revlay"
fi

# 清理临时文件
rm -rf "${TEMP_DIR}"

echo ""
echo "🎉 Revlay 已成功安装！"
echo "请运行 'revlay --help' 来开始使用。"