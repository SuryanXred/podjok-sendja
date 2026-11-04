package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/database"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/persistence"
	"caffe-podjok-sendja/internal/view"
)

const sessionCookieName = "pds_customer_session"

func getLoggedInCustomer(c *gin.Context) *customer.Customer {
	token, err := c.Cookie(sessionCookieName)
	if err != nil || token == "" {
		return nil
	}
	return customer.DefaultStore.GetCustomerBySession(token)
}

func setCustomerSession(c *gin.Context, cust *customer.Customer) {
	sess, err := customer.DefaultStore.CreateSession(cust.ID)
	if err != nil {
		return
	}
	c.SetCookie(
		sessionCookieName,
		sess.Token,
		7*86400, // 7 hari
		"/",
		"",
		false,
		true,
	)
}

func clearCustomerSession(c *gin.Context) {
	token, _ := c.Cookie(sessionCookieName)
	if token != "" {
		customer.DefaultStore.DeleteSession(token)
	}
	c.SetCookie(
		sessionCookieName,
		"",
		-1,
		"/",
		"",
		false,
		true,
	)
}

func loginPageHandler(c *gin.Context) {
	if cust := getLoggedInCustomer(c); cust != nil {
		c.Redirect(http.StatusSeeOther, "/akun")
		return
	}

	var page bytes.Buffer
	if err := view.LoginPage("", "").Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat halaman login.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func loginPostHandler(c *gin.Context) {
	identifier := strings.TrimSpace(c.PostForm("identifier"))
	password := c.PostForm("password")

	cust, err := customer.DefaultStore.Authenticate(identifier, password)
	if err != nil {
		var page bytes.Buffer
		_ = view.LoginPage(err.Error(), "").Render(c.Request.Context(), &page)
		c.Data(http.StatusUnprocessableEntity, "text/html; charset=utf-8", page.Bytes())
		return
	}

	setCustomerSession(c, cust)

	// Jika ada barang di keranjang, arahkan ke keranjang agar siap checkout
	cart := getCart(c)
	if !cart.IsEmpty() {
		c.Redirect(http.StatusSeeOther, "/keranjang")
		return
	}

	c.Redirect(http.StatusSeeOther, "/akun")
}

func registerPageHandler(c *gin.Context) {
	if cust := getLoggedInCustomer(c); cust != nil {
		c.Redirect(http.StatusSeeOther, "/akun")
		return
	}

	initialType := strings.TrimSpace(c.Query("type"))
	if initialType != "umum" {
		initialType = "mahasiswa"
	}

	var page bytes.Buffer
	if err := view.RegisterPage("", initialType).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat halaman pendaftaran.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func registerPostHandler(c *gin.Context) {
	rawType := strings.TrimSpace(c.PostForm("type"))
	cType := customer.CustomerType(rawType)
	nim := strings.TrimSpace(c.PostForm("nim"))
	name := strings.TrimSpace(c.PostForm("name"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")

	cust, err := customer.DefaultStore.Register(customer.RegisterInput{
		Type:     cType,
		Name:     name,
		Phone:    phone,
		Email:    email,
		NIM:      nim,
		Password: password,
	})

	if err != nil {
		var page bytes.Buffer
		_ = view.RegisterPage(err.Error(), rawType).Render(c.Request.Context(), &page)
		c.Data(http.StatusUnprocessableEntity, "text/html; charset=utf-8", page.Bytes())
		return
	}

	setCustomerSession(c, cust)
	if err := database.SaveCustomer(cust); err != nil {
		log.Printf("[register] gagal menyimpan pelanggan %s ke MySQL: %v", cust.ID, err)
	}
	persistence.SaveCustomers()

	// Jika ada barang di keranjang, langsung arahkan ke keranjang
	cart := getCart(c)
	if !cart.IsEmpty() {
		c.Redirect(http.StatusSeeOther, "/keranjang")
		return
	}

	c.Redirect(http.StatusSeeOther, "/akun")
}

func logoutHandler(c *gin.Context) {
	clearCustomerSession(c)
	c.Redirect(http.StatusSeeOther, "/menu")
}

func accountPageHandler(c *gin.Context) {
	cust := getLoggedInCustomer(c)
	if cust == nil {
		c.Redirect(http.StatusSeeOther, "/masuk")
		return
	}

	orders := order.DefaultStore.ListOrdersByCustomerID(cust.ID)

	var page bytes.Buffer
	if err := view.AccountPage(cust, orders).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat halaman akun.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}
