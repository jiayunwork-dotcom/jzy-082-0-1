// Package api 提供轮胎纵向力计算服务的 HTTP 接口层（Gin）。
// 接口层只做 JSON 编解码与参数存在性检查，所有计算委托给 tire 包。
package api

import (
	"github.com/gin-gonic/gin"
)

// NewRouter 构建服务的 HTTP 路由。无任何共享可变状态，
// 并发请求之间天然隔离。
func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	v1 := r.Group("/v1")
	v1.POST("/force", handleForce)
	v1.POST("/peak", handlePeak)
	v1.POST("/curve", handleCurve)
	v1.GET("/demo", handleDemo)

	return r
}
