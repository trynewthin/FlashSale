// Package logdir 统一管理 cmd/fs 运行时日志目录。
//
// 约束：
// - `.memory/` 仅用于本地记忆与规划文件，不应在部署环境作为运行日志目录。
// - 运行日志默认写入仓库根目录下的 `log/`，便于容器挂载与运维查看。
package logdir

import "path/filepath"

// Root 返回仓库级运行日志根目录（绝对路径）。
func Root(repoRoot string) string {
	return filepath.Join(repoRoot, "log")
}

// ServicesDir 返回服务进程日志目录（stdout/stderr/pid）。
func ServicesDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "services")
}

// FrontendsDir 返回前端 dev server 日志目录。
func FrontendsDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "frontends")
}

// OpsJobsDir 返回 ops-control 任务日志归档目录。
func OpsJobsDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "ops-jobs")
}

// DataDir 返回数据工具输出目录（seed 结果等）。
func DataDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "data")
}
