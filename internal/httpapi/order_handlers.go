package httpapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/database"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/payment"
	"caffe-podjok-sendja/internal/persistence"
	"caffe-podjok-sendja/internal/view"
)

func homeHandler(c *gin.Context) {
	c.Redirect(http.StatusSeeOther, "/menu")
}

func cartAddHandler(c *gin.Context) {
	slug := strings.TrimSpace(c.PostForm("product_slug"))
	variant := strings.TrimSpace(c.PostForm("variant"))

	if slug != "" {
		addItemToCart(c, slug, variant, 1)
	}

	referer := c.Request.Header.Get("Referer")
	if strings.Contains(referer, "/keranjang") {
		c.Redirect(http.StatusSeeOther, "/keranjang")
		return
	}
	c.Redirect(http.StatusSeeOther, "/menu")
}

func cartUpdateHandler(c *gin.Context) {
	indexStr := strings.TrimSpace(c.PostForm("index"))
	deltaStr := strings.TrimSpace(c.PostForm("delta"))

	index, err1 := strconv.Atoi(indexStr)
	delta, err2 := strconv.Atoi(deltaStr)

	if err1 == nil && err2 == nil {
		updateItemQtyInCart(c, index, delta)
	}

	c.Redirect(http.StatusSeeOther, "/keranjang")
}

func cartRemoveHandler(c *gin.Context) {
	indexStr := strings.TrimSpace(c.PostForm("index"))
	index, err := strconv.Atoi(indexStr)
	if err == nil {
		removeItemFromCart(c, index)
	}

	c.Redirect(http.StatusSeeOther, "/keranjang")
}

func cartPageHandler(c *gin.Context) {
	cart := getCart(c)
	cust := getLoggedInCustomer(c)

	var page bytes.Buffer
	if err := view.CartPage(cart, cust, "").Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat keranjang.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func checkoutHandler(c *gin.Context) {
	cart := getCart(c)
	if cart.IsEmpty() {
		c.Redirect(http.StatusSeeOther, "/keranjang")
		return
	}

	cust := getLoggedInCustomer(c)

	customerName := strings.TrimSpace(c.PostForm("customer_name"))
	customerPhone := strings.TrimSpace(c.PostForm("customer_phone"))
	orderType := order.OrderType(strings.TrimSpace(c.PostForm("order_type")))
	tableNumber := strings.TrimSpace(c.PostForm("table_number"))
	notes := strings.TrimSpace(c.PostForm("notes"))
	paymentMethod := order.PaymentMethod(strings.TrimSpace(c.PostForm("payment_method")))

	var discount int64
	var discountNote string
	var customerID, customerType, customerNIM string

	if cust != nil {
		customerID = cust.ID
		customerType = string(cust.Type)
		customerNIM = cust.NIM
		discount, discountNote = cust.CalculateFirstDiscount(cart.Subtotal)
	}

	ord, err := order.DefaultStore.CreateOrder(order.CreateOrderInput{
		CustomerID:    customerID,
		CustomerType:  customerType,
		CustomerNIM:   customerNIM,
		CustomerName:  customerName,
		CustomerPhone: customerPhone,
		OrderType:     orderType,
		TableNumber:   tableNumber,
		Notes:         notes,
		PaymentMethod: paymentMethod,
		Discount:      discount,
		DiscountNote:  discountNote,
		Cart:          cart,
	})

	if err != nil {
		var page bytes.Buffer
		_ = view.CartPage(cart, cust, err.Error()).Render(c.Request.Context(), &page)
		c.Data(http.StatusUnprocessableEntity, "text/html; charset=utf-8", page.Bytes())
		return
	}

	if cust != nil && discount > 0 {
		_ = customer.DefaultStore.MarkFirstDiscountUsed(cust.ID)
		persistence.SaveCustomers()
	}

	if ord.PaymentMethod == order.PaymentMethodQRIS {
		if qrisRes, err := payment.GenerateQRIS(ord.Code, ord.Total); err == nil {
			order.DefaultStore.SetOrderQRIS(ord.Code, qrisRes.QRURL, qrisRes.QRString)
		}
	}

	_ = database.SaveOrder(ord)
	persistence.SaveOrders()
	persistence.SaveCatalog()

	clearCartCookie(c)
	c.Redirect(http.StatusSeeOther, "/pesanan/"+ord.Code)
}

func orderPageHandler(c *gin.Context) {
	code := c.Param("kode")
	cust := getLoggedInCustomer(c)
	ord, err := order.DefaultStore.GetOrderByCode(code)
	if err != nil {
		var page bytes.Buffer
		_ = view.TrackPage("Pesanan dengan kode \""+code+"\" tidak ditemukan.", cust).Render(c.Request.Context(), &page)
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", page.Bytes())
		return
	}

	if ord.PaymentMethod == order.PaymentMethodQRIS && ord.QRISURL == "" && ord.OrderStatus == order.OrderStatusPendingPayment {
		if qrisRes, err := payment.GenerateQRIS(ord.Code, ord.Total); err == nil {
			order.DefaultStore.SetOrderQRIS(ord.Code, qrisRes.QRURL, qrisRes.QRString)
		}
	}

	var page bytes.Buffer
	if err := view.OrderPage(ord, cust).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat nota pesanan.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func publicPrintReceiptHandler(c *gin.Context) {
	code := strings.TrimSpace(c.Param("kode"))
	ord, err := order.DefaultStore.GetOrderByCode(code)
	if err != nil {
		c.String(http.StatusNotFound, "Pesanan tidak ditemukan.")
		return
	}

	staffUser := getLoggedInStaff(c)

	var page bytes.Buffer
	if err := view.ReceiptPrintPage(ord, staffUser).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat struk cetak: "+err.Error())
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func trackPageHandler(c *gin.Context) {
	cust := getLoggedInCustomer(c)
	var page bytes.Buffer
	if err := view.TrackPage("", cust).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat halaman lacak pesanan.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func trackSearchHandler(c *gin.Context) {
	code := strings.TrimSpace(c.PostForm("code"))
	cust := getLoggedInCustomer(c)
	if code == "" {
		var page bytes.Buffer
		_ = view.TrackPage("Silakan masukkan kode pesanan.", cust).Render(c.Request.Context(), &page)
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", page.Bytes())
		return
	}

	ord, err := order.DefaultStore.GetOrderByCode(code)
	if err != nil {
		var page bytes.Buffer
		_ = view.TrackPage("Pesanan dengan kode \""+code+"\" tidak ditemukan.", cust).Render(c.Request.Context(), &page)
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", page.Bytes())
		return
	}

	c.Redirect(http.StatusSeeOther, "/pesanan/"+ord.Code)
}

func orderCancelHandler(c *gin.Context) {
	code := c.Param("kode")
	if err := order.DefaultStore.CancelOrder(code); err != nil {
		cust := getLoggedInCustomer(c)
		var page bytes.Buffer
		_ = view.TrackPage("Gagal membatalkan pesanan: "+err.Error(), cust).Render(c.Request.Context(), &page)
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", page.Bytes())
		return
	}
	_ = database.UpdateOrderStatus(code, string(order.OrderStatusCancelled), string(order.PaymentStatusUnpaid))
	persistence.SaveOrders()
	persistence.SaveCatalog()
	c.Redirect(http.StatusSeeOther, "/pesanan/"+strings.ToUpper(strings.TrimSpace(code)))
}

func midtransWebhookHandler(c *gin.Context) {
	notif, err := payment.ParseNotification(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := payment.GetConfig()
	if !payment.VerifySignature(*notif, cfg.ServerKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature tidak valid"})
		return
	}

	switch notif.TransactionStatus {
	case "capture", "settlement":
		_ = order.DefaultStore.UpdateOrderStatus(notif.OrderID, order.OrderStatusPaid, order.PaymentStatusPaid)
		_ = database.UpdateOrderStatus(notif.OrderID, string(order.OrderStatusPaid), string(order.PaymentStatusPaid))
		persistence.SaveOrders()
		persistence.SaveCatalog()
	case "cancel", "expire", "deny":
		_ = order.DefaultStore.UpdateOrderStatus(notif.OrderID, order.OrderStatusCancelled, order.PaymentStatusUnpaid)
		_ = database.UpdateOrderStatus(notif.OrderID, string(order.OrderStatusCancelled), string(order.PaymentStatusUnpaid))
		persistence.SaveOrders()
		persistence.SaveCatalog()
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func simulateQRISPaymentHandler(c *gin.Context) {
	code := c.Param("kode")
	_ = order.DefaultStore.UpdateOrderStatus(code, order.OrderStatusPaid, order.PaymentStatusPaid)
	_ = database.UpdateOrderStatus(code, string(order.OrderStatusPaid), string(order.PaymentStatusPaid))
	persistence.SaveOrders()
	persistence.SaveCatalog()
	c.Redirect(http.StatusSeeOther, "/pesanan/"+strings.ToUpper(strings.TrimSpace(code)))
}

func orderStatusAPIHandler(c *gin.Context) {
	code := c.Param("kode")
	ord, err := order.DefaultStore.GetOrderByCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pesanan tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":           ord.Code,
		"order_status":   ord.OrderStatus,
		"payment_status": ord.PaymentStatus,
		"is_paid":        ord.PaymentStatus == order.PaymentStatusPaid,
	})
}
