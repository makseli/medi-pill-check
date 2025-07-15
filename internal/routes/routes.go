package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/makseli/medi-pill-check/internal/config"
	"github.com/makseli/medi-pill-check/internal/handlers"
	"github.com/makseli/medi-pill-check/internal/middleware"
)

func Setup(r *gin.Engine, h *handlers.Handlers) {
	r.GET("/", h.Health)

	api := r.Group("/api")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)

		// Auth gerektiren endpointler
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware(config.Load()))
		{
			auth.GET("/users", h.ListUsers)
			auth.GET("/users/:id", h.GetUser)
			auth.PUT("/users/:id", h.UpdateUser)
			auth.DELETE("/users/:id", h.DeleteUser)
		}
	}
}
