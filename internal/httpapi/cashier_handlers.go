package httpapi

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/database"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/persistence"
	"caffe-podjok-sendja/internal/view"
)

func cashierPageHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/kasir")
		return
	}

	statusQuery := strings.TrimSpace(c.Query("status"))
	filterStatus := order.OrderStatus(statusQuery)
	searchQuery := strings.TrimSpace(c.Query("q"))

	metrics := order.DefaultStore.GetCashierMetrics()
	orders := order.DefaultStore.SearchOrders(searchQuery, filterStatus)

	var page bytes.Buffer
	if err := view.CashierPage(metrics, orders, filterStatus, searchQuery, staffUser).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat antrean kasir.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func cashierUpdateOrderStatusHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/kasir")
		return
	}

	code := c.Param("kode")
	newStatus := order.OrderStatus(strings.TrimSpace(c.PostForm("status")))
	paymentStatus := order.PaymentStatus(strings.TrimSpace(c.PostForm("payment_status")))

	if newStatus != "" {
		if paymentStatus == "" {
			if newStatus == order.OrderStatusPaid || newStatus == order.OrderStatusPreparing || newStatus == order.OrderStatusReady || newStatus == order.OrderStatusCompleted {
				paymentStatus = order.PaymentStatusPaid
			} else {
				paymentStatus = order.PaymentStatusUnpaid
			}
		}
		_ = order.DefaultStore.UpdateOrderStatus(code, newStatus, paymentStatus)
		_ = database.UpdateOrderStatus(code, string(newStatus), string(paymentStatus))
		persistence.SaveOrders()
		persistence.SaveCatalog()
	}

	referer := c.Request.Header.Get("Referer")
	if referer != "" && (strings.Contains(referer, "/kasir") || strings.Contains(referer, "/admin/pesanan")) {
		c.Redirect(http.StatusSeeOther, referer)
		return
	}
	c.Redirect(http.StatusSeeOther, "/kasir")
}

func cashierPrintReceiptHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/kasir")
		return
	}

	code := strings.TrimSpace(c.Param("kode"))
	ord, err := order.DefaultStore.GetOrderByCode(code)
	if err != nil {
		c.String(http.StatusNotFound, "Pesanan tidak ditemukan.")
		return
	}

	var page bytes.Buffer
	if err := view.ReceiptPrintPage(ord, staffUser).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat struk cetak: "+err.Error())
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}
