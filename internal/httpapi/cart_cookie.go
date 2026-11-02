package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/order"
)

const cartCookieName = "pds_cart"

type rawCartItem struct {
	ProductSlug string `json:"product_slug"`
	Variant     string `json:"variant"`
	Quantity    int    `json:"quantity"`
}

func readRawCart(c *gin.Context) []rawCartItem {
	cookieVal, err := c.Cookie(cartCookieName)
	if err != nil || cookieVal == "" {
		return nil
	}

	data, err := base64.URLEncoding.DecodeString(cookieVal)
	if err != nil {
		return nil
	}

	var items []rawCartItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil
	}
	return items
}

func writeRawCart(c *gin.Context, items []rawCartItem) {
	if len(items) == 0 {
		clearCartCookie(c)
		return
	}

	data, err := json.Marshal(items)
	if err != nil {
		return
	}

	encoded := base64.URLEncoding.EncodeToString(data)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cartCookieName, encoded, 86400*7, "/", "", false, true)
}

func clearCartCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cartCookieName, "", -1, "/", "", false, true)
}

func getCart(c *gin.Context) order.Cart {
	raw := readRawCart(c)
	var converted []struct {
		ProductSlug string `json:"product_slug"`
		Variant     string `json:"variant"`
		Quantity    int    `json:"quantity"`
	}
	for _, r := range raw {
		converted = append(converted, struct {
			ProductSlug string `json:"product_slug"`
			Variant     string `json:"variant"`
			Quantity    int    `json:"quantity"`
		}{
			ProductSlug: r.ProductSlug,
			Variant:     r.Variant,
			Quantity:    r.Quantity,
		})
	}
	return order.BuildCart(converted)
}

func addItemToCart(c *gin.Context, slug, variant string, qty int) {
	if qty <= 0 {
		qty = 1
	}

	items := readRawCart(c)
	found := false
	for i := range items {
		if items[i].ProductSlug == slug && items[i].Variant == variant {
			items[i].Quantity += qty
			found = true
			break
		}
	}
	if !found {
		items = append(items, rawCartItem{
			ProductSlug: slug,
			Variant:     variant,
			Quantity:    qty,
		})
	}
	writeRawCart(c, items)
}

func updateItemQtyInCart(c *gin.Context, index int, delta int) {
	items := readRawCart(c)
	if index < 0 || index >= len(items) {
		return
	}

	items[index].Quantity += delta
	if items[index].Quantity <= 0 {
		items = append(items[:index], items[index+1:]...)
	}
	writeRawCart(c, items)
}

func removeItemFromCart(c *gin.Context, index int) {
	items := readRawCart(c)
	if index < 0 || index >= len(items) {
		return
	}
	items = append(items[:index], items[index+1:]...)
	writeRawCart(c, items)
}
