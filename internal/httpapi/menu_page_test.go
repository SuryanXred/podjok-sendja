package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		amount int64
		want   string
	}{
		{0, "Rp 0"},
		{4000, "Rp 4.000"},
		{15000, "Rp 15.000"},
		{20000, "Rp 20.000"},
		{125000, "Rp 125.000"},
		{1000000, "Rp 1.000.000"},
	}

	for _, tt := range tests {
		if got := formatRupiah(tt.amount); got != tt.want {
			t.Errorf("formatRupiah(%d) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

func fetchMenuPage(t *testing.T) string {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := NewRouter()
	request := httptest.NewRequest(http.MethodGet, "/menu", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("content type = %q, want text/html", contentType)
	}
	return response.Body.String()
}

func TestMenuPageRendersCatalog(t *testing.T) {
	body := fetchMenuPage(t)

	want := []string{
		"Podjok Sendja",
		"Coffee",
		"Americano",
		"Rp 15.000",
		"Rp 12.000",
		"Tea &amp; Milk",
		"French Fries",
		"Rp 20.000",
	}
	for _, text := range want {
		if !strings.Contains(body, text) {
			t.Errorf("page does not contain %q", text)
		}
	}
}

func TestMenuPageHasOnlyWorkingNavigation(t *testing.T) {
	body := fetchMenuPage(t)

	for _, link := range []string{`href="/menu"`, `href="/keranjang"`, `href="/lacak"`} {
		if !strings.Contains(body, link) {
			t.Errorf("page has no link to active route %q", link)
		}
	}

	restrictedOrDeadLinks := []string{`href="/kasir"`, `href="/admin"`, `href="/tentang"`, `href="/kontak"`}
	for _, link := range restrictedOrDeadLinks {
		if strings.Contains(body, link) {
			t.Errorf("public page should not contain internal or dead link %q", link)
		}
	}
}

func TestMenuPageUsesSenjaShell(t *testing.T) {
	body := fetchMenuPage(t)

	want := []string{
		`id="top"`,
		`class="site-table"`,
		`class="hero"`,
		`class="hero-kicker"`,
		`class="hero-title"`,
		`class="menu-sheet"`,
		`class="back-to-top"`,
		`href="#top"`,
		`name="theme-color" content="#2A1D17"`,
	}
	for _, text := range want {
		if !strings.Contains(body, text) {
			t.Errorf("page does not contain %q", text)
		}
	}

	if !strings.Contains(body, "<img") {
		t.Error("page must display category and product photos")
	}
}

func TestMenuPageNumbersCategories(t *testing.T) {
	body := fetchMenuPage(t)

	for _, number := range []string{"01", "02", "03", "04"} {
		if !strings.Contains(body, `class="menu-index">`+number+`<`) {
			t.Errorf("page does not number a category with %q", number)
		}
	}
}

func TestMenuPageAssetsLoad(t *testing.T) {
	body := fetchMenuPage(t)
	router := NewRouter()

	assets := []string{
		"/static/app.css",
		"/static/favicon.svg",
		"/static/images/products/americano.webp",
		"/static/images/categories/Coffee.jpeg",
		"/static/images/site/hero.webp",
	}
	for _, asset := range assets {
		if !strings.Contains(body, asset) {
			t.Errorf("page does not reference asset %q", asset)
		}

		request := httptest.NewRequest(http.MethodGet, asset, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Errorf("%s status = %d, want %d", asset, response.Code, http.StatusOK)
		}
		if response.Body.Len() == 0 {
			t.Errorf("%s returned an empty body", asset)
		}
	}
}
