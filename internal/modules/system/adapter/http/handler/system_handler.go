package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/pkg/health"
)

// SystemHandler 处理系统级 HTTP 端点。
type SystemHandler struct {
	healthRegistry *health.Registry
}

// NewSystemHandler 创建系统 handler。
func NewSystemHandler(registry *health.Registry) *SystemHandler {
	return &SystemHandler{healthRegistry: registry}
}

// Healthz 存活探针——进程在即可返回 200。
func (h *SystemHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz 就绪探针——检查所有依赖的健康状态。
func (h *SystemHandler) Readyz(c *gin.Context) {
	result := h.healthRegistry.RunAll(c.Request.Context())
	status := http.StatusOK
	if result.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, result)
}
