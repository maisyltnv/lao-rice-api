// Seed sample rice categories, products, and default admin (idempotent).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"shopapi/internal/config"
	"shopapi/internal/database"
	"shopapi/internal/model"
	"shopapi/internal/repository"
	"shopapi/internal/service"
	"shopapi/seed"

	"gorm.io/gorm"
)

func main() {
	adminOnly := flag.Bool("admin-only", false, "only ensure default admin user")
	flag.Parse()

	cfg := config.Load()
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	ctx := context.Background()
	if err := seed.EnsureDefaultAdmin(ctx, db); err != nil {
		log.Fatalf("admin seed: %v", err)
	}
	if *adminOnly {
		log.Println("admin seed completed")
		return
	}
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	categorySvc := service.NewCategoryService(categoryRepo, productRepo)
	productSvc := service.NewProductService(productRepo, categoryRepo)

	type catSeed struct {
		slug, name, desc string
		sort             int
	}
	cats := []catSeed{
		{slug: "khao-jao", name: "ເຂົ້າຈ້າວ", desc: "ເຂົ້າຈ້າວຫອມ ແລະ ເຂົ້າຈ້າວພິເສດ", sort: 1},
		{slug: "khao-niao", name: "ເຂົ້າໜຽວ", desc: "ເຂົ້າໜຽວຄຸນນະພາບດີ ສຳລັບຄົນລາວ", sort: 2},
	}

	catID := map[string]uint64{}
	for _, c := range cats {
		id, err := ensureCategory(ctx, db, categorySvc, c.slug, c.name, c.desc, c.sort)
		if err != nil {
			log.Fatalf("category %s: %v", c.name, err)
		}
		catID[c.slug] = id
		log.Printf("category: %s (id=%d)", c.name, id)
	}

	type prodSeed struct {
		name, desc, catSlug, imageURL string
		priceLak                     float64
		weightKg                     float64
	}
	// Paths match lao-rice-web/public/images/rice (web resolves /images/...; app uses bundled assets).
	const img = "/images/rice"
	products := []prodSeed{
		{name: "ເຂົ້ານາປີ", desc: "ເຂົ້າໜຽວນາປີ ຫົວກ້ຽວ 25 ກິໂລ", catSlug: "khao-niao", priceLak: 185_000, weightKg: 25, imageURL: img + "/grains.jpg"},
		{name: "ເຂົ້ານາແຊງ", desc: "ເຂົ້າໜຽວນາແຊງ ຫົວຍາວ 25 ກິໂລ", catSlug: "khao-niao", priceLak: 195_000, weightKg: 25, imageURL: img + "/sticky.jpg"},
		{name: "ເຂົ້າໄກ່ນ້ອຍ", desc: "ເຂົ້າໜຽວໄກ່ນ້ອຍ 25 ກິໂລ", catSlug: "khao-niao", priceLak: 175_000, weightKg: 25, imageURL: img + "/bowl.jpg"},
		{name: "ເຂົ້າຈ້າວມະລິ", desc: "ເຂົ້າຈ້າວມະລິ ຫອມນຸ່ມ 25 ກິໂລ", catSlug: "khao-jao", priceLak: 220_000, weightKg: 25, imageURL: img + "/field.jpg"},
		{name: "ເຂົ້າເຈົ້າໄຮ່", desc: "ເຂົ້າຈ້າວເຈົ້າໄຮ່ 25 ກິໂລ", catSlug: "khao-jao", priceLak: 210_000, weightKg: 25, imageURL: img + "/bag.jpg"},
	}

	for _, p := range products {
		cid := catID[p.catSlug]
		categoryID := cid
		if err := ensureProduct(ctx, db, productSvc, p.name, p.desc, p.imageURL, &categoryID, p.priceLak, p.weightKg); err != nil {
			log.Fatalf("product %s: %v", p.name, err)
		}
		log.Printf("product: %s — %s LAK", p.name, fmt.Sprintf("%.0f", p.priceLak))
	}

	log.Println("seed completed (products + admin)")
}

func ensureCategory(ctx context.Context, db *gorm.DB, svc *service.CategoryService, slug, name, desc string, sort int) (uint64, error) {
	var existing model.Category
	err := db.WithContext(ctx).Where("slug = ?", slug).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return 0, err
	}
	active := true
	cat, err := svc.Create(ctx, service.CreateCategoryInput{
		Name:        name,
		Slug:        slug,
		Description: desc,
		SortOrder:   sort,
		IsActive:    &active,
	})
	if err != nil {
		return 0, err
	}
	return cat.ID, nil
}

func ensureProduct(
	ctx context.Context,
	db *gorm.DB,
	svc *service.ProductService,
	name, desc, imageURL string,
	categoryID *uint64,
	priceLak, weightKg float64,
) error {
	var existing model.Product
	err := db.WithContext(ctx).Where("name = ?", name).First(&existing).Error
	if err == nil {
		return db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"description":        desc,
			"image_url":          imageURL,
			"category_id":        categoryID,
			"final_price_lak":    priceLak,
			"weight_kg":          weightKg,
			"original_price_cny": 0,
			"exchange_rate":      1,
			"profit_margin":      0,
		}).Error
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	final := priceLak
	_, err = svc.Create(ctx, service.CreateProductInput{
		Name:             name,
		Description:      desc,
		ImageURL:         imageURL,
		CategoryID:       categoryID,
		OriginalPriceCNY: 0,
		ExchangeRate:     1,
		ProfitMargin:     0,
		FinalPriceLAK:    &final,
	})
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Model(&model.Product{}).Where("name = ?", name).
		Update("weight_kg", weightKg).Error
}
