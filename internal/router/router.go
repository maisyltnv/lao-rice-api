package router

import (
	"net/http"

	"shopapi/internal/handler"
	"shopapi/internal/middleware"
	"shopapi/internal/service"

	"github.com/gin-gonic/gin"
)

// New wires HTTP routes: public catalog, JWT-protected mutations, auth, and orders.
func New(
	auth *service.AuthService,
	authH *handler.AuthHandler,
	categoryH *handler.CategoryHandler,
	productH *handler.ProductHandler,
	orderH *handler.OrderHandler,
	exchangeH *handler.ExchangeRateHandler,
	bannerH *handler.BannerHandler,
	uploadDir string,
	imagesDir string,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if reqh := c.GetHeader("Access-Control-Request-Headers"); reqh != "" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", reqh)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		}
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if uploadDir != "" {
		r.Static("/uploads", uploadDir)
	}
	if imagesDir != "" {
		r.Static("/images", imagesDir)
	}

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", authH.Register)
		authGroup.POST("/login", authH.Login)
		authGroup.POST("/otp/send", authH.SendOTP)
		authGroup.POST("/otp/verify", authH.VerifyOTP)
		authGroup.GET("/me", middleware.JWTAuth(auth), authH.Me)

		authGroup.POST("/admin/register", authH.AdminRegister)
		authGroup.POST("/admin/login", authH.AdminLogin)
		authGroup.GET("/admin/me", middleware.JWTAuth(auth), middleware.RequireAdmin(), authH.AdminMe)
	}

	r.GET("/categories", categoryH.List)
	r.GET("/categories/slug/:slug", categoryH.GetBySlug)
	r.GET("/categories/:id", categoryH.Get)

	r.GET("/products", productH.List)
	r.GET("/products/:id", productH.Get)

	r.GET("/exchange-rate", exchangeH.Get)

	r.GET("/banners", middleware.OptionalJWTAuth(auth), bannerH.List)
	r.GET("/banners/:id", middleware.OptionalJWTAuth(auth), bannerH.Get)

	r.GET("/orders/shipping-config", orderH.ShippingConfig)
	r.GET("/orders/shipping-quote", orderH.QuoteShipping)
	r.GET("/ordersbyphone", orderH.ListByPhone)

	protected := r.Group("")
	protected.Use(middleware.JWTAuth(auth))
	{
		protected.POST("/orders", orderH.Place)
		protected.POST("/categories", categoryH.Create)
		protected.PUT("/categories/:id", categoryH.Update)
		protected.DELETE("/categories/:id", categoryH.Delete)

		protected.POST("/products", productH.Create)
		protected.PUT("/products/:id", productH.Update)
		protected.DELETE("/products/:id", productH.Delete)

		protected.GET("/orders", middleware.RequireAdmin(), orderH.List)
		protected.GET("/orders/:id/source-links", middleware.RequireAdmin(), orderH.GetSourceLinks)
		protected.GET("/orders/:id", orderH.Get)
		protected.PUT("/orders/:id/status", middleware.RequireAdmin(), orderH.UpdateStatus)

		protected.PUT("/exchange-rate", middleware.RequireAdmin(), exchangeH.Set)

		protected.POST("/banners", middleware.RequireAdmin(), bannerH.Create)
		protected.PUT("/banners/:id", middleware.RequireAdmin(), bannerH.Update)
		protected.DELETE("/banners/:id", middleware.RequireAdmin(), bannerH.Delete)
	}

	return r
}
