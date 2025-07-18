#!/bin/bash

# 确保脚本在任何命令失败时立即退出
set -e

# ==================================
#    加载 .env 文件中的环境变量
# ==================================
# 获取脚本所在的目录，从而定位项目根目录
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROJECT_ROOT=$(dirname "$SCRIPT_DIR")
ENV_FILE="$PROJECT_ROOT/.env"

# 加载 .env 文件
if [ -f "$ENV_FILE" ]; then
  # 使用 set -a 来自动导出从 .env 文件中 source 的所有变量
  set -a
  source "$ENV_FILE"
  set +a
fi


# ==================================
#      Revlay 交互式发布脚本
# ==================================

# 检查依赖项
if ! command -v gum &> /dev/null; then
    echo "错误: 'gum' 未安装。请运行 'brew install gum'。"
    exit 1
fi
if ! command -v goreleaser &> /dev/null; then
    echo "错误: 'goreleaser' 未安装。请运行 'brew install goreleaser'。"
    exit 1
fi

# 在脚本早期就检查 GITHUB_TOKEN
if [ -z "$GITHUB_TOKEN" ]; then
    gum style --border normal --margin "1" --padding "1 2" --border-foreground 99 \
        "错误: GITHUB_TOKEN 未设置。" \
        "请在项目根目录的 .env 文件中提供此变量，" \
        "或者手动导出: export GITHUB_TOKEN=your_token"
    exit 1
fi

# 检查 Git 工作目录是否干净
if ! git diff-index --quiet HEAD --; then
    gum style --border normal --margin "1" --padding "1 2" --border-foreground 212 \
        "Git 工作目录不干净。" \
        "请在发布前提交或暂存您的改动。"
    exit 1
fi

# 通过管道将文本传递给 gum style，避免被误认为 flag
echo '--- 🚀 Revlay 发布流程启动 ---' | gum style --foreground 212

# 1. 选择版本类型
gum style '选择版本更新类型:'
VERSION_TYPE=$(gum choose "patch" "minor" "major" "prerelease")

# 2. 计算并确认版本号
# 获取最新的 git tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
# 移除 'v' 前缀
LAST_VERSION=${LAST_TAG#v}

# 使用IFS分割版本号
IFS='.' read -r -a V_PARTS <<< "$LAST_VERSION"
MAJOR=${V_PARTS[0]}
MINOR=${V_PARTS[1]}
# 处理 beta/rc 标签
PATCH=$(echo "${V_PARTS[2]}" | cut -d- -f1)

case "$VERSION_TYPE" in
    "patch")
        PATCH=$((PATCH + 1))
        ;;
    "minor")
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    "major")
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
esac

SUGGESTED_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

if [ "$VERSION_TYPE" = "prerelease" ]; then
    PRE_RELEASE_LABEL=$(gum input --placeholder "例如: beta.1, rc.1")
    SUGGESTED_VERSION="${SUGGESTED_VERSION}-${PRE_RELEASE_LABEL}"
fi

gum style "建议的版本号: ${SUGGESTED_VERSION}"
VERSION=$(gum input --value "$SUGGESTED_VERSION" --placeholder "请输入最终版本号...")

# 3. 编写版本说明
gum style '请输入版本标题 (例如: "新增 XYZ 功能"):'
TITLE=$(gum input --placeholder "版本标题")

gum style '请输入详细的更新说明 (Ctrl+D 保存并退出):'
DESCRIPTION=$(gum write --placeholder "在这里详细描述更新内容...")

# 4. 最终确认
# 使用 here-document 将多行文本赋值给变量，更清晰、安全
CONFIRM_DETAILS=$(cat <<EOF
即将执行以下操作:
  - 版本: ${VERSION}
  - 标题: ${TITLE}
  - Git 推送: main 分支及新标签
  - 发布到 GitHub Releases
EOF
)
gum style --border normal --margin "1" --padding "1 2" --border-foreground 212 "$CONFIRM_DETAILS"

if ! gum confirm "是否继续?"; then
    gum style --foreground 212 "发布已取消。"
    exit 0
fi

# 5. 执行发布流程
echo
echo '--- 正在提交改动... ---' | gum style --bold --foreground 212
git commit --allow-empty -m "chore(release): Release ${VERSION}"

echo
echo '--- 正在创建 Git 标签... ---' | gum style --bold --foreground 212
git tag -a "$VERSION" -m "$TITLE"$'\n\n'"$DESCRIPTION"

echo
echo '--- 正在推送代码和标签到远程仓库... ---' | gum style --bold --foreground 212
git push
git push origin "$VERSION"

echo
echo '--- 正在使用 GoReleaser 发布... ---' | gum style --bold --foreground 212
goreleaser release --clean

echo
echo "🎉 发布完成！版本 ${VERSION} 已成功发布到 GitHub Releases！" | gum style --bold --foreground 212