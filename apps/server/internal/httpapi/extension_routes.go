package httpapi

import (
	"github.com/gin-gonic/gin"

	"lirs/apps/server/internal/store"
)

func registerExtensionRoutes(api *gin.RouterGroup, repo repository) {
	api.GET("/training/courses", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.TrainingCourses(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/courses", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingCourseInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingCourse(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/courses/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingCourseInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingCourse(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/authorizations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.TrainingAuthorizations(c.Request.Context())
		if err == nil && !canManageTraining(actor) {
			filtered := make([]store.TrainingAuthorization, 0, len(item))
			for _, record := range item {
				if record.UserID == actor.UserID || record.UserName == actor.Name {
					filtered = append(filtered, record)
				}
			}
			item = filtered
		}
		respond(c, item, err)
	})
	api.POST("/training/authorizations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.TrainingAuthorizationInput
		if bindJSON(c, &input) {
			if !isAdmin(actor) {
				input.UserID = actor.UserID
				input.UserName = actor.Name
				input.Status = "pending"
			}
			if input.UserID == "" {
				input.UserID = actor.UserID
			}
			if input.UserName == "" {
				input.UserName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveTrainingAuthorization(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/authorizations/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingAuthorizationInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingAuthorization(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/questions", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.TrainingQuestions(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/questions", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingQuestionInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingQuestion(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/questions/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingQuestionInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingQuestion(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/exams", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		item, err := repo.TrainingExams(c.Request.Context())
		if err == nil && !isAdmin(actor) {
			filtered := make([]store.TrainingExam, 0, len(item))
			for _, record := range item {
				if record.UserID == actor.UserID || record.UserName == actor.Name {
					filtered = append(filtered, record)
				}
			}
			item = filtered
		}
		respond(c, item, err)
	})
	api.POST("/training/exams", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.TrainingExamInput
		if bindJSON(c, &input) {
			if !isAdmin(actor) {
				input.UserID = actor.UserID
				input.UserName = actor.Name
			}
			if input.UserName == "" {
				input.UserName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveTrainingExam(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/exams/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingExamInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingExam(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/practicals", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.TrainingPracticals(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/practicals", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingPracticalInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingPractical(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/practicals/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingPracticalInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingPractical(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/training/rules", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, materialAdminRoles()...); !ok {
			return
		}
		item, err := repo.TrainingRules(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/training/rules", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingRuleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingRule(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/training/rules/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.TrainingRuleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveTrainingRule(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/workflows/:kind", func(c *gin.Context) {
		if _, ok := requireAnyRole(c, repo, tenantAdminRoles()...); !ok {
			return
		}
		item, err := repo.BusinessConfigs(c.Request.Context(), "workflow", c.Param("kind"))
		respond(c, item, err)
	})
	api.POST("/workflows/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(c.Request.Context(), "workflow", c.Param("kind"), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/workflows/:kind/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(c.Request.Context(), "workflow", c.Param("kind"), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/billing/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		item, err := repo.BusinessConfigs(ctx, "billing", c.Param("kind"))
		respond(c, item, err)
	})
	api.POST("/billing/:kind", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(ctx, "billing", c.Param("kind"), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/billing/:kind/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, financeAdminRoles()...)
		if !ok {
			return
		}
		ctx, ok := financeRequestContext(c, repo, actor)
		if !ok {
			return
		}
		var input store.BusinessConfigInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveBusinessConfig(ctx, "billing", c.Param("kind"), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/spaces", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.Spaces(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/spaces", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.SpaceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSpace(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/spaces/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.SpaceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSpace(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/space-reservations", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.SpaceReservations(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/space-reservations", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.SpaceReservationInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			input.Requester = actor.Name
			input.Actor = actor.Name
			item, err := repo.CreateSpaceReservation(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/samples", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.Samples(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/samples", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.SampleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSample(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/samples/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.SampleInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveSample(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/sample-movements", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.SampleMovements(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/sample-movements", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, materialAdminRoles()...)
		if !ok {
			return
		}
		var input store.SampleMovementInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.CreateSampleMovement(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
	api.GET("/lims/tasks", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.LimsTasks(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/lims/tasks", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.LimsTaskInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			if input.RequesterName == "" {
				input.RequesterName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveLimsTask(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/lims/tasks/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.LimsTaskInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveLimsTask(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/eln/records", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.ElnRecords(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/eln/records", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.ElnRecordInput
		if bindJSON(c, &input) {
			input.AuthorID = actor.UserID
			if input.AuthorName == "" {
				input.AuthorName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveElnRecord(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/eln/records/:id", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.ElnRecordInput
		if bindJSON(c, &input) {
			input.AuthorID = actor.UserID
			if input.AuthorName == "" {
				input.AuthorName = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.SaveElnRecord(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/iot/devices", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.IotDevices(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/iot/devices", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.IotDeviceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveIotDevice(c.Request.Context(), "", input)
			respond(c, item, err)
		}
	})
	api.PATCH("/iot/devices/:id", func(c *gin.Context) {
		actor, ok := requireAnyRole(c, repo, tenantAdminRoles()...)
		if !ok {
			return
		}
		var input store.IotDeviceInput
		if bindJSON(c, &input) {
			input.Actor = actor.Name
			item, err := repo.SaveIotDevice(c.Request.Context(), c.Param("id"), input)
			respond(c, item, err)
		}
	})
	api.GET("/ai-assistant", func(c *gin.Context) {
		if _, ok := requireActiveUser(c, repo); !ok {
			return
		}
		item, err := repo.AssistantQueries(c.Request.Context())
		respond(c, item, err)
	})
	api.POST("/ai-assistant", func(c *gin.Context) {
		actor, ok := requireActiveUser(c, repo)
		if !ok {
			return
		}
		var input store.AssistantQueryInput
		if bindJSON(c, &input) {
			input.RequesterID = actor.UserID
			if input.Requester == "" {
				input.Requester = actor.Name
			}
			input.Actor = actor.Name
			item, err := repo.AskAssistant(c.Request.Context(), input)
			respond(c, item, err)
		}
	})
}
