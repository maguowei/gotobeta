package localid

import (
	"context"
	"errors"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Snowflake 变体的位分配：41 位毫秒时间戳 + 10 位节点号 + 12 位序列号，共 63 位，正好落在正 int64 范围内。
const (
	// epoch 是自定义纪元（2024-01-01 UTC，毫秒），缩短时间戳位宽以延长可用年限（约 69 年）。
	epoch int64 = 1704067200000

	nodeBits     uint8 = 10
	sequenceBits uint8 = 12

	maxNodeID   int64 = -1 ^ (-1 << nodeBits)     // 1023
	maxSequence int64 = -1 ^ (-1 << sequenceBits) // 4095

	timeShift uint8 = nodeBits + sequenceBits // 22
	nodeShift uint8 = sequenceBits            // 12
)

// Generator 生成进程内单调、跨副本唯一的 int64 ID。
//
// 老实现 time.Now().UnixMilli()*1000 + seq%1000 在水平扩展下会冲突：各副本 seq 都从 0 起，
// 同一毫秒不同副本会产生完全相同的 ID，作为数据库主键写入时主键冲突。Snowflake 通过引入
// 节点号位把不同副本的 ID 空间隔离开，避免该冲突。
type Generator struct {
	mu       sync.Mutex
	nodeID   int64
	lastMs   int64
	sequence int64
}

// New 创建本地 ID 生成器。节点号取自 APP_NODE_ID（0-1023），缺省时由主机名派生，
// 确保多副本部署各副本节点号不同。生产可显式设置 APP_NODE_ID 保证唯一性。
func New() *Generator {
	return &Generator{nodeID: resolveNodeID()}
}

func resolveNodeID() int64 {
	if raw := strings.TrimSpace(os.Getenv("APP_NODE_ID")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n >= 0 {
			return n & maxNodeID
		}
	}
	// 退化：用主机名哈希派生节点号。k8s 下各 Pod 主机名唯一，足以把不同副本散列到不同节点号，
	// 显著降低同毫秒主键冲突概率。
	if host, err := os.Hostname(); err == nil && host != "" {
		h := fnv.New32a()
		_, _ = h.Write([]byte(host))
		return int64(h.Sum32()) & maxNodeID
	}

	return 0
}

// NextID 生成下一个 ID。同一毫秒内序列号自增；序列号用尽时自旋等待下一毫秒。
func (g *Generator) NextID(context.Context) (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()
	if now < g.lastMs {
		// 时钟回拨：拒绝生成，避免产生小于已发号的 ID 造成冲突。
		return 0, errors.New("localid: 时钟回拨，拒绝生成 ID")
	}

	if now == g.lastMs {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			// 当前毫秒序列号用尽，等待进入下一毫秒。
			now = waitNextMilli(g.lastMs)
		}
	} else {
		g.sequence = 0
	}
	g.lastMs = now

	id := ((now - epoch) << timeShift) | (g.nodeID << nodeShift) | g.sequence

	return id, nil
}

func waitNextMilli(last int64) int64 {
	now := time.Now().UnixMilli()
	for now <= last {
		now = time.Now().UnixMilli()
	}

	return now
}
