package handler

import "github.com/gin-gonic/gin"

func RegisterAPI(public, protected, staff, admin *gin.RouterGroup, movies *MovieHandler, authHandler *AuthHandler, users *UserHandler) {
	public.POST("/auth/register", authHandler.Register)
	public.POST("/auth/login", authHandler.Login)
	public.GET("/movies", movies.list)
	public.GET("/movies/", movies.getByID)
	public.GET("/movies/:id", movies.getByID)

	protected.GET("/me", users.me)

	staff.POST("/movies", movies.create)
	staff.PUT("/movies/:id", movies.replace)

	admin.DELETE("/movies/:id", movies.delete)
	admin.GET("/users", users.list)
	admin.PUT("/users/:id/role", users.updateRole)
}
