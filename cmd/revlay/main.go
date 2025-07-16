package main

import (
	"github.com/xukonxe/revlay/internal/cli"
)

// 这个变量将由 GoReleaser 注入
var version string

func main() {
	// 将版本号传递给 cli 包
	cli.SetVersion(version)
	cli.Execute()
}
