package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type menuVariant struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

type menuProduct struct {
	Slug     string        `json:"slug"`
	Name     string        `json:"name"`
	Price    int64         `json:"price"`
	Variants []menuVariant `json:"variants"`
}

type menuCategory struct {
	Slug     string        `json:"slug"`
	Name     string        `json:"name"`
	Products []menuProduct `json:"products"`
}

type menuResponse struct {
	Categories []menuCategory `json:"categories"`
}

func fetchMenu(t *testing.T) menuResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := NewRouter()
	request := httptest.NewRequest(http.MethodGet, "/api/menu", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q, want application/json", contentType)
	}

	var body menuResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func TestMenuEndpointReturnsFourCategories(t *testing.T) {
	body := fetchMenu(t)

	if len(body.Categories) != 4 {
		t.Fatalf("category count = %d, want 4", len(body.Categories))
	}

	want := []string{"Coffee", "Tea & Milk", "Food", "Snack"}
	for i, name := range want {
		if body.Categories[i].Name != name {
			t.Errorf("category %d = %q, want %q", i, body.Categories[i].Name, name)
		}
	}
}

func TestMenuEndpointReturnsVariantPrices(t *testing.T) {
	body := fetchMenu(t)

	americano := findProduct(t, body, "americano")
	if americano.Price != 12000 {
		t.Errorf("Americano base price = %d, want 12000", americano.Price)
	}
	if len(americano.Variants) != 2 {
		t.Fatalf("Americano variant count = %d, want 2", len(americano.Variants))
	}

	tests := []menuVariant{
		{Name: "Ice", Price: 15000},
		{Name: "Hot", Price: 12000},
	}
	for i, want := range tests {
		if americano.Variants[i] != want {
			t.Errorf("variant %d = %+v, want %+v", i, americano.Variants[i], want)
		}
	}
}

func TestMenuEndpointReturnsSinglePriceProducts(t *testing.T) {
	body := fetchMenu(t)

	espresso := findProduct(t, body, "espresso")
	if espresso.Price != 10000 {
		t.Errorf("Espresso price = %d, want 10000", espresso.Price)
	}
	if len(espresso.Variants) != 0 {
		t.Errorf("Espresso variant count = %d, want 0", len(espresso.Variants))
	}

	platter := findProduct(t, body, "mix-platter")
	if platter.Price != 20000 {
		t.Errorf("Mix Platter price = %d, want 20000", platter.Price)
	}
}

func findProduct(t *testing.T, body menuResponse, slug string) menuProduct {
	t.Helper()

	for _, category := range body.Categories {
		for _, product := range category.Products {
			if product.Slug == slug {
				return product
			}
		}
	}
	t.Fatalf("product %q not found", slug)
	return menuProduct{}
}
