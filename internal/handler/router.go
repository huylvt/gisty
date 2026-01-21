package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/huylvt/gisty/internal/config"
	"github.com/huylvt/gisty/internal/middleware"
	"github.com/huylvt/gisty/internal/repository"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RouterDeps contains dependencies for the router
type RouterDeps struct {
	PasteHandler *PasteHandler
	RateLimiter  *middleware.RateLimiter
	S3Client     *repository.S3
}

// NewRouter creates and configures a new Gin router
func NewRouter(cfg *config.Config, deps *RouterDeps) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Swagger documentation
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check and API routes (require deps)
	if deps != nil {
		// Health check
		healthHandler := NewHealthHandler(deps.S3Client)
		router.GET("/health", healthHandler.Health)
		router.GET("/debug/s3", healthHandler.DebugS3)
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Paste routes
		if deps != nil && deps.PasteHandler != nil {
			// Apply content size limit and rate limiting to POST endpoint
			postMiddlewares := []gin.HandlerFunc{
				middleware.ContentSizeMiddleware(),
			}
			if deps.RateLimiter != nil {
				postMiddlewares = append(postMiddlewares, deps.RateLimiter.Middleware())
			}
			postMiddlewares = append(postMiddlewares, deps.PasteHandler.CreatePaste)
			v1.POST("/pastes", postMiddlewares...)

			v1.GET("/pastes/:id", deps.PasteHandler.GetPaste)
			v1.DELETE("/pastes/:id", deps.PasteHandler.DeletePaste)
		}
	}

	// Short URL route (must be after API routes to avoid conflicts)
	if deps != nil && deps.PasteHandler != nil {
		router.GET("/:id", deps.PasteHandler.ShortURL)
	}

	return router
}

// corsMiddleware returns a configured CORS middleware
func corsMiddleware() gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "X-Syntax-Type", "X-Created-At", "X-Expires-At", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: false,
		MaxAge:           12 * 60 * 60, // 12 hours
	}
	return cors.New(config)
}