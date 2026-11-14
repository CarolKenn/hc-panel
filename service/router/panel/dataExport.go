package panel

import (
	"sun-panel/api/api_v1"
	"sun-panel/api/api_v1/middleware"

	"github.com/gin-gonic/gin"
)

// InitDataExport 数据导出（备份）路由，仅登录用户可访问自己的数据
func InitDataExport(router *gin.RouterGroup) {
	api := api_v1.ApiGroupApp.ApiPanel.DataExport
	r := router.Group("", middleware.LoginInterceptor)
	{
		r.POST("/panel/dataExport/getSummary", api.GetSummary)
		r.POST("/panel/dataExport/export", api.Export)
	}
}
