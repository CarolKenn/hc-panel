package panel

import (
	"sun-panel/api/api_v1"
	"sun-panel/api/api_v1/middleware"

	"github.com/gin-gonic/gin"
)

// InitDataImport 数据导入（恢复）路由，仅登录用户可操作自己的数据
func InitDataImport(router *gin.RouterGroup) {
	api := api_v1.ApiGroupApp.ApiPanel.DataImport
	r := router.Group("", middleware.LoginInterceptor)
	{
		r.POST("/panel/dataImport/preview", api.Preview)
		r.POST("/panel/dataImport/apply", api.Apply)
	}
}
