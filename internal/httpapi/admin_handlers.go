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
