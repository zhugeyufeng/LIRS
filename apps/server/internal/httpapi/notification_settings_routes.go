package httpapi

import (
	"github.com/gin-gonic/gin"

	"lirs/apps/server/internal/store"
)

func registerGraphMailRoutes(api *gin.RouterGroup, repo repository) {
	api.PATCH("/notification-channel-settings/graph-mail", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.GraphMailSettingsInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveGraphMailSettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.POST("/notification-channel-settings/graph-mail/test", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, "super_admin")
		if !ok {
			return
		}
		var input store.GraphMailTestInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.TestGraphMailSettings(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
}

func registerDingTalkNotificationRoutes(api *gin.RouterGroup, repo repository) {
	api.POST("/notification-channel-settings/dingtalk/test", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles...)
		if !ok {
			return
		}
		requestCtx, ok := tenantAdminRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.DingTalkTestInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.TestDingTalkSettings(requestCtx, input)
			respond(c, item, err)
		}
	})
}
