package httpapi

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/catalog"
	"caffe-podjok-sendja/internal/database"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/persistence"
	"caffe-podjok-sendja/internal/storeconfig"
	"caffe-podjok-sendja/internal/view"
)

func adminDashboardHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	// Otorisasi: Hanya Admin yang dapat mengakses Dashboard Admin!
	if !staffUser.IsAdmin() {
		var page bytes.Buffer
		_ = view.StaffForbiddenPage(staffUser).Render(c.Request.Context(), &page)
		c.Data(http.StatusForbidden, "text/html; charset=utf-8", page.Bytes())
		return
	}

	tab := strings.TrimSpace(c.Query("tab"))
	if tab == "" {
		tab = "laporan"
	}
	msg := strings.TrimSpace(c.Query("msg"))

	report := order.DefaultStore.GetAdminReport()
	categories := catalog.DefaultStore.GetCategories()
	stockList := catalog.DefaultStore.GetStockList()
	movements := catalog.DefaultStore.GetStockMovements(15)
	cfg := storeconfig.Get()

	var page bytes.Buffer
	if err := view.AdminDashboardPage(tab, report, categories, stockList, movements, cfg, msg, staffUser).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat dashboard admin: "+err.Error())
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func adminAddProductHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil || !staffUser.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	categorySlug := strings.TrimSpace(c.PostForm("category_slug"))
	name := strings.TrimSpace(c.PostForm("name"))
	priceStr := strings.TrimSpace(c.PostForm("base_price"))
	variantsStr := strings.TrimSpace(c.PostForm("variants"))

	basePrice, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil || basePrice <= 0 {
		c.Redirect(http.StatusSeeOther, "/admin?tab=menu&msg="+url.QueryEscape("Harga dasar menu tidak valid."))
		return
	}

	var variants []catalog.Variant
	if variantsStr != "" {
		parts := strings.Split(variantsStr, ",")
		for _, p := range parts {
			item := strings.TrimSpace(p)
			if item == "" {
				continue
			}
			subParts := strings.Split(item, ":")
			varName := strings.TrimSpace(subParts[0])
			varDelta := int64(0)
			if len(subParts) > 1 {
				if d, err := strconv.ParseInt(strings.TrimSpace(subParts[1]), 10, 64); err == nil {
					varDelta = d
				}
			}
			if varName != "" {
				variants = append(variants, catalog.Variant{
					Name:       varName,
					PriceDelta: varDelta,
				})
			}
		}
	}

	prod, err := catalog.DefaultStore.AddProduct(categorySlug, name, basePrice, variants)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin?tab=menu&msg="+url.QueryEscape("Gagal menambah menu: "+err.Error()))
		return
	}

	if prod != nil {
		_ = database.AddProduct(categorySlug, prod, 20)
	}
	persistence.SaveCatalog()
	c.Redirect(http.StatusSeeOther, "/admin?tab=menu&msg="+url.QueryEscape("Menu '"+name+"' berhasil ditambahkan ke katalog."))
}

func adminUpdatePriceHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil || !staffUser.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	slug := strings.TrimSpace(c.PostForm("product_slug"))
	priceStr := strings.TrimSpace(c.PostForm("new_price"))

	newPrice, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil || newPrice <= 0 {
		c.Redirect(http.StatusSeeOther, "/admin?tab=menu&msg="+url.QueryEscape("Harga baru tidak valid."))
		return
	}

	if err := catalog.DefaultStore.UpdateProductPrice(slug, newPrice); err != nil {
		c.Redirect(http.StatusSeeOther, "/admin?tab=menu&msg="+url.QueryEscape("Gagal memperbarui harga: "+err.Error()))
		return
	}

	_ = database.UpdateProductPrice(slug, newPrice)
	persistence.SaveCatalog()
	c.Redirect(http.StatusSeeOther, "/admin?tab=menu&msg="+url.QueryEscape("Harga menu berhasil diperbarui menjadi "+view.FormatRupiah(newPrice)+"."))
}

func adminAdjustStockHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil || !staffUser.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	slug := strings.TrimSpace(c.PostForm("product_slug"))
	actionType := strings.TrimSpace(c.PostForm("action_type"))
	qtyStr := strings.TrimSpace(c.PostForm("quantity"))
	reason := strings.TrimSpace(c.PostForm("reason"))

	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty <= 0 {
		c.Redirect(http.StatusSeeOther, "/admin?tab=stok&msg="+url.QueryEscape("Jumlah porsi stok harus lebih besar dari 0."))
		return
	}

	change := qty
	if actionType == "sub" {
		change = -qty
	}

	if err := catalog.DefaultStore.AdjustStock(slug, change, reason); err != nil {
		c.Redirect(http.StatusSeeOther, "/admin?tab=stok&msg="+url.QueryEscape("Gagal menyesuaikan stok: "+err.Error()))
		return
	}

	currentStock, _ := catalog.DefaultStore.GetStock(slug)
	_ = database.AdjustStock(slug, currentStock, change, reason)
	persistence.SaveCatalog()
	c.Redirect(http.StatusSeeOther, "/admin?tab=stok&msg="+url.QueryEscape("Stok menu berhasil diperbarui."))
}

func adminUpdateSettingsHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil || !staffUser.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	isOpen := c.PostForm("is_open") == "true"
	hours := strings.TrimSpace(c.PostForm("opening_hours"))
	phone := strings.TrimSpace(c.PostForm("contact_phone"))
	tablesStr := strings.TrimSpace(c.PostForm("table_count"))
	announcement := strings.TrimSpace(c.PostForm("announcement"))
	address := strings.TrimSpace(c.PostForm("address"))

	tables, _ := strconv.Atoi(tablesStr)
	if tables <= 0 {
		tables = 15
	}

	newCfg := storeconfig.Update(isOpen, hours, phone, tables, announcement, address)
	_ = database.UpdateStoreConfig(newCfg)
	persistence.SaveConfig()
	c.Redirect(http.StatusSeeOther, "/admin?tab=pengaturan&msg="+url.QueryEscape("Pengaturan operasional kafe berhasil disimpan."))
}

// adminOrdersPageHandler redirects to the dedicated cashier panel at /kasir for seamless compatibility
func adminOrdersPageHandler(c *gin.Context) {
	target := "/kasir"
	if c.Request.URL.RawQuery != "" {
		target += "?" + c.Request.URL.RawQuery
	}
	c.Redirect(http.StatusMovedPermanently, target)
}

func adminUpdateOrderStatusHandler(c *gin.Context) {
	cashierUpdateOrderStatusHandler(c)
}

func adminExportSalesReportCSVHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil || !staffUser.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	period := strings.TrimSpace(c.Query("period"))
	if period == "" {
		period = "all"
	}
	status := order.OrderStatus(strings.TrimSpace(c.Query("status")))

	orders := order.DefaultStore.ListOrdersFiltered(period, status)

	timestamp := time.Now().Format("20060102-1504")
	filename := fmt.Sprintf("laporan-penjualan-caffe-podjok-sendja-%s-%s.csv", period, timestamp)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")

	if err := generateOrdersCSV(c.Writer, orders); err != nil {
		c.String(http.StatusInternalServerError, "Gagal membuat berkas CSV: "+err.Error())
		return
	}
}

func generateOrdersCSV(w io.Writer, orders []*order.Order) error {
	// Tulis UTF-8 BOM agar Microsoft Excel membuka karakter khusus tanpa rusak
	if _, err := w.Write([]byte("\xef\xbb\xbf")); err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := []string{
		"Kode Pesanan",
		"Waktu Transaksi (WITA)",
		"Nama Pelanggan",
		"No. Telepon",
		"Kategori Pelanggan",
		"NIM Mahasiswa",
		"Tipe Pesanan",
		"Nomor Meja",
		"Metode Pembayaran",
		"Status Pembayaran",
		"Status Pesanan",
		"Rincian Menu",
		"Total Porsi",
		"Subtotal (Rp)",
		"Diskon (Rp)",
		"Keterangan Diskon",
		"Total Bayar (Rp)",
		"Catatan Pesanan",
	}

	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, o := range orders {
		var itemDetails []string
		totalPorsi := 0
		for _, it := range o.Items {
			name := it.ProductName
			if it.Variant != "" {
				name += " (" + it.Variant + ")"
			}
			itemDetails = append(itemDetails, fmt.Sprintf("%s x%d", name, it.Quantity))
			totalPorsi += it.Quantity
		}

		custType := "Umum"
		if o.CustomerType == "mahasiswa" {
			custType = "Mahasiswa"
		}

		tableNo := "-"
		if o.TableNumber != "" {
			tableNo = o.TableNumber
		}

		record := []string{
			o.Code,
			o.CreatedAt.Format("02/01/2006 15:04:05"),
			o.CustomerName,
			o.CustomerPhone,
			custType,
			o.CustomerNIM,
			o.OrderTypeLabel(),
			tableNo,
			o.PaymentMethodLabel(),
			o.PaymentStatusLabel(),
			o.StatusLabel(),
			strings.Join(itemDetails, "; "),
			fmt.Sprintf("%d", totalPorsi),
			fmt.Sprintf("%d", o.Subtotal),
			fmt.Sprintf("%d", o.Discount),
			o.DiscountNote,
			fmt.Sprintf("%d", o.Total),
			o.Notes,
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// adminExportStockMovementsCSVHandler mengekspor riwayat mutasi pergerakan stok ke berkas CSV (UTF-8 BOM).
func adminExportStockMovementsCSVHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil || !staffUser.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/pegawai/masuk?redirect=/admin")
		return
	}

	movements := catalog.DefaultStore.GetStockMovements(0)
	timestamp := time.Now().Format("20060102-1504")
	filename := fmt.Sprintf("riwayat-mutasi-stok-caffe-podjok-sendja-%s.csv", timestamp)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")

	if _, err := c.Writer.Write([]byte("\xef\xbb\xbf")); err != nil {
		c.String(http.StatusInternalServerError, "Gagal menulis BOM CSV: "+err.Error())
		return
	}

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	headers := []string{
		"ID Mutasi",
		"Waktu Mutasi (WITA)",
		"Kode Menu (Slug)",
		"Nama Menu Produk",
		"Perubahan Stok (Porsi)",
		"Jenis Mutasi",
		"Alasan / Keterangan Mutasi",
	}

	if err := writer.Write(headers); err != nil {
		c.String(http.StatusInternalServerError, "Gagal membuat header CSV: "+err.Error())
		return
	}

	for _, m := range movements {
		signChange := fmt.Sprintf("%d", m.Change)
		changeType := "Keluar / Pengurangan (-)"
		if m.Change > 0 {
			signChange = "+" + signChange
			changeType = "Masuk / Penambahan (+)"
		}

		record := []string{
			fmt.Sprintf("%d", m.ID),
			m.CreatedAt.Format("02/01/2006 15:04:05"),
			m.ProductSlug,
			m.ProductName,
			signChange,
			changeType,
			m.Reason,
		}

		if err := writer.Write(record); err != nil {
			return
		}
	}
}

// adminLiveMonitoringAPIHandler menyediakan data real-time JSON untuk pemantauan penjualan dan stok.
func adminLiveMonitoringAPIHandler(c *gin.Context) {
	staffUser := getLoggedInStaff(c)
	if staffUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Staff session expired or unauthorized"})
		return
	}
	if !staffUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya Administrator yang memiliki akses ke pemantauan real-time"})
		return
	}

	report := order.DefaultStore.GetAdminReport()
	stockList := catalog.DefaultStore.GetStockList()
	movements := catalog.DefaultStore.GetStockMovements(30)

	type jsonOrder struct {
		Code           string `json:"code"`
		CustomerName   string `json:"customer_name"`
		CustomerType   string `json:"customer_type"`
		TableNumber    string `json:"table_number"`
		StatusLabel    string `json:"status_label"`
		StatusCode     string `json:"status_code"`
		TotalFormatted string `json:"total_formatted"`
		Total          int64  `json:"total"`
		TimeFormatted  string `json:"time_formatted"`
		ItemsSummary   string `json:"items_summary"`
	}

	recentOrdersJSON := make([]jsonOrder, 0, len(report.RecentOrders))
	for _, o := range report.RecentOrders {
		items := make([]string, 0, len(o.Items))
		for _, it := range o.Items {
			name := it.ProductName
			if it.Variant != "" {
				name += " (" + it.Variant + ")"
			}
			items = append(items, fmt.Sprintf("%s x%d", name, it.Quantity))
		}
		recentOrdersJSON = append(recentOrdersJSON, jsonOrder{
			Code:           o.Code,
			CustomerName:   o.CustomerName,
			CustomerType:   o.CustomerType,
			TableNumber:    o.TableNumber,
			StatusLabel:    o.StatusLabel(),
			StatusCode:     string(o.OrderStatus),
			TotalFormatted: view.FormatRupiah(o.Total),
			Total:          o.Total,
			TimeFormatted:  o.CreatedAt.Format("15:04"),
			ItemsSummary:   strings.Join(items, ", "),
		})
	}

	type jsonTopProduct struct {
		Name           string `json:"name"`
		Quantity       int    `json:"quantity"`
		SalesFormatted string `json:"sales_formatted"`
	}

	topProductsJSON := make([]jsonTopProduct, 0, len(report.TopProducts))
	for _, p := range report.TopProducts {
		topProductsJSON = append(topProductsJSON, jsonTopProduct{
			Name:           p.ProductName,
			Quantity:       p.Quantity,
			SalesFormatted: view.FormatRupiah(p.TotalSales),
		})
	}

	type jsonStockItem struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Category    string `json:"category"`
		Qty         int    `json:"qty"`
		StatusLabel string `json:"status_label"`
		StatusCode  string `json:"status_code"`
	}

	stockJSON := make([]jsonStockItem, 0, len(stockList))
	safeCount, lowCount, emptyCount := 0, 0, 0
	for _, it := range stockList {
		switch it.StatusCode() {
		case "safe":
			safeCount++
		case "low":
			lowCount++
		case "empty":
			emptyCount++
		}
		stockJSON = append(stockJSON, jsonStockItem{
			Slug:        it.ProductSlug,
			Name:        it.ProductName,
			Category:    it.CategoryName,
			Qty:         it.Qty,
			StatusLabel: it.StatusLabel(),
			StatusCode:  it.StatusCode(),
		})
	}

	type jsonMovement struct {
		ID            int    `json:"id"`
		TimeFormatted string `json:"time_formatted"`
		ProductName   string `json:"product_name"`
		Change        int    `json:"change"`
		Reason        string `json:"reason"`
	}

	movementsJSON := make([]jsonMovement, 0, len(movements))
	for _, m := range movements {
		movementsJSON = append(movementsJSON, jsonMovement{
			ID:            m.ID,
			TimeFormatted: m.CreatedAt.Format("02/01/2006, 15:04 WIB"),
			ProductName:   m.ProductName,
			Change:        m.Change,
			Reason:        m.Reason,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().Format("02/01/2006 15:04:05 WITA"),
		"metrics": gin.H{
			"confirmed_sales":     view.FormatRupiah(report.Metrics.ConfirmedSales),
			"confirmed_sales_raw": report.Metrics.ConfirmedSales,
			"total_orders":        report.Metrics.TotalOrders,
			"completed_orders":    report.Metrics.Completed,
			"pending_orders":      report.Metrics.PendingPayment,
			"average_order_value": view.FormatRupiah(report.AverageOrderValue),
			"mahasiswa_orders":    report.MahasiswaOrders,
			"umum_orders":         report.UmumOrders,
			"cash_orders":         report.CashOrders,
			"qris_orders":         report.QRISOrders,
		},
		"recent_orders": recentOrdersJSON,
		"top_products":  topProductsJSON,
		"stock_summary": gin.H{
			"safe_count":  safeCount,
			"low_count":   lowCount,
			"empty_count": emptyCount,
			"total_items": len(stockList),
		},
		"stock_items":     stockJSON,
		"stock_movements": movementsJSON,
	})
}
