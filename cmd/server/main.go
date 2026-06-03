package main

import (
	"log"
	"net/http"

	"shopapi/internal/config"
	"shopapi/internal/database"
	"shopapi/internal/handler"
	"shopapi/internal/repository"
	"shopapi/internal/router"
	"shopapi/internal/service"
	"shopapi/internal/upload"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	gin.SetMode(gin.ReleaseMode)

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	exchangeRepo := repository.NewExchangeRateRepository(db)
	bannerRepo := repository.NewBannerRepository(db)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiryH)
	otpSvc := service.NewOTPService(cfg.OTPStubCode, cfg.OTPExpiryMinutes)
	categorySvc := service.NewCategoryService(categoryRepo, productRepo)
	productSvc := service.NewProductService(productRepo, categoryRepo)
	orderSvc := service.NewOrderService(orderRepo, productRepo, cfg.ShippingFeeLAK, cfg.FreeShippingMinSubtotalLAK)
	exchangeSvc := service.NewExchangeRateService(exchangeRepo, productRepo)
	bannerSvc := service.NewBannerService(bannerRepo)

	authH := handler.NewAuthHandler(authSvc, otpSvc)
	categoryH := handler.NewCategoryHandler(categorySvc)
	productH := handler.NewProductHandler(productSvc)
	receiptStore, err := upload.NewPaymentReceiptStore(cfg.UploadDir, cfg.UploadURLPrefix)
	if err != nil {
		log.Fatalf("uploads: %v", err)
	}

	orderH := handler.NewOrderHandler(orderSvc, receiptStore)
	exchangeH := handler.NewExchangeRateHandler(exchangeSvc)
	bannerH := handler.NewBannerHandler(bannerSvc)

	r := router.New(authSvc, authH, categoryH, productH, orderH, exchangeH, bannerH, cfg.UploadDir, cfg.ImagesDir)

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
