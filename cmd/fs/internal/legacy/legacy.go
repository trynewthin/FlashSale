// Package legacy 提供“兼容旧版本目录结构”的开关。
//
// 背景：
// - 早期版本会把运行日志写到 `.memory/runlogs`。
// - 部署环境不应依赖 `.memory/`（该目录仅用于本地记忆/规划文件）。
//
// 规则：
// - 默认不启用 legacy（也就不会读取/展示 `.memory/runlogs`）。
// - 仅在显式设置环境变量 `FLASHSALE_ALLOW_LEGACY_MEMORY=true` 时才启用。
package legacy

import "os"

// AllowLegacyMemory 返回是否允许读取/展示旧的 `.memory/runlogs` 目录内容。
func AllowLegacyMemory() bool {
	switch os.Getenv("FLASHSALE_ALLOW_LEGACY_MEMORY") {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	default:
		return false
	}
}
