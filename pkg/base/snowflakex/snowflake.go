// snowflakex 提供 Snowflake 节点初始化辅助函数，用于容器环境下自动分配唯一节点 ID。
package snowflakex

import (
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/snowflake"
)

// maxNodeBits 与 snowflake 默认 NodeBits=10 对齐，nodeID 范围 0~1023。
const maxNodeBits = 1023

// NewNode 创建 Snowflake 节点，解析顺序：
//  1. 环境变量 SNOWFLAKE_NODE（如果设为整数）
//  2. cfgNode（配置文件中的 SnowflakeNode）
//  3. 用 hostname hash 自动生成（Docker Compose 副本的 hostname 不同，保证唯一性）
func NewNode(cfgNode int64) (*snowflake.Node, error) {
	nodeID := resolveNodeID(cfgNode)
	return snowflake.NewNode(nodeID)
}

func resolveNodeID(cfgNode int64) int64 {
	// 优先级 1: 环境变量
	if env := strings.TrimSpace(os.Getenv("SNOWFLAKE_NODE")); env != "" {
		if v, err := strconv.ParseInt(env, 10, 64); err == nil && v >= 0 {
			return v & maxNodeBits
		}
	}
	// 优先级 2: 配置文件值 + hostname hash offset
	// 将 hostname 的 hash 混入 cfgNode，使同一服务不同容器获得不同 nodeID
	hostname, _ := os.Hostname()
	if hostname == "" {
		if cfgNode > 0 {
			return cfgNode & maxNodeBits
		}
		return 1
	}
	h := fnvHash(hostname)
	if cfgNode > 0 {
		// 用 cfgNode 的高位 + hostname hash 的低位组合
		return ((cfgNode << 5) | (h & 0x1f)) & maxNodeBits
	}
	return h & maxNodeBits
}

// fnvHash 计算字符串的 FNV-1a 32bit hash 并返回 int64。
func fnvHash(s string) int64 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int64(h)
}
