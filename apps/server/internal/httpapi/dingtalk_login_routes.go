package httpapi

import (
	"github.com/gin-gonic/gin"

	"lirs/apps/server/internal/store"
)

func registerDingTalkLoginRoutes(api *gin.RouterGroup, repo repository) {
	api.POST("/dingtalk/web-login-intent", func(c *gin.Context) {
		var input store.DingTalkWebLoginIntentInput
		if bindJSON(c, &input) {
			item, err := repo.DingTalkWebLoginIntent(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/dingtalk/web-login", func(c *gin.Context) {
		var input store.DingTalkWebLoginInput
		if bindJSON(c, &input) {
			item, err := repo.DingTalkWebLogin(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/dingtalk/login-bind-existing", func(c *gin.Context) {
		var input store.DingTalkLoginBindExistingInput
		if bindJSON(c, &input) {
			item, err := repo.BindDingTalkLoginToExistingUser(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
}
