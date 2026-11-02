package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/catalog"
)

type variantDTO struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

type productDTO struct {
	Slug     string       `json:"slug"`
	Name     string       `json:"name"`
	Price    int64        `json:"price"`
	Variants []variantDTO `json:"variants"`
}

type categoryDTO struct {
	Slug     string       `json:"slug"`
	Name     string       `json:"name"`
	Products []productDTO `json:"products"`
}

type menuDTO struct {
	Categories []categoryDTO `json:"categories"`
}

func menuHandler(c *gin.Context) {
	categories := catalog.Seed()
	payload := menuDTO{Categories: make([]categoryDTO, 0, len(categories))}

	for _, category := range categories {
		categoryItem := categoryDTO{
			Slug:     category.Slug,
			Name:     category.Name,
			Products: make([]productDTO, 0, len(category.Products)),
		}

		for _, product := range category.Products {
			productItem := productDTO{
				Slug:     product.Slug,
				Name:     product.Name,
				Price:    product.BasePrice,
				Variants: make([]variantDTO, 0, len(product.Variants)),
			}

			for _, variant := range product.Variants {
				productItem.Variants = append(productItem.Variants, variantDTO{
					Name:  variant.Name,
					Price: product.BasePrice + variant.PriceDelta,
				})
			}

			categoryItem.Products = append(categoryItem.Products, productItem)
		}

		payload.Categories = append(payload.Categories, categoryItem)
	}

	c.JSON(http.StatusOK, payload)
}
