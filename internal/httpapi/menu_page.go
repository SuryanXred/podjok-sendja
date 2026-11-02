package httpapi

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/catalog"
	"caffe-podjok-sendja/internal/view"
)

func menuPageHandler(c *gin.Context) {
	cart := getCart(c)
	cust := getLoggedInCustomer(c)
	var page bytes.Buffer
	if err := view.MenuPage(catalog.Seed(), cart, cust).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Menu sedang tidak dapat ditampilkan. Coba muat ulang halaman.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}
