package health

import (
	"context"
	"time"
)

// DBHealthResult 数据库健康检查结果
type DBHealthResult struct {
	Healthy      bool  `json:"healthy"`
	ResponseTime int64 `json:"responseTime"`
	Error        string `json:"error,omitempty"`
}

// HealthService 健康检查服务接口
type HealthService interface {
	CheckDatabase(ctx context.Context) DBHealthResult
}

// PluginHealthResult 插件健康检查结果
type PluginHealthResult struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Healthy bool   `json:"healthy"`
	Duration time.Duration `json:"-"`
}