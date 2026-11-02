package httpapi

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	assets, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	router.StaticFS("/static", http.FS(assets))

	router.GET("/", homeHandler)
	router.GET("/menu", menuPageHandler)
	router.GET("/api/menu", menuHandler)

	router.POST("/cart/add", cartAddHandler)
	router.POST("/cart/update", cartUpdateHandler)
	router.POST("/cart/remove", cartRemoveHandler)
	router.GET("/keranjang", cartPageHandler)

	router.POST("/checkout", checkoutHandler)
	router.GET("/pesanan/:kode", orderPageHandler)
	router.GET("/pesanan/:kode/cetak", publicPrintReceiptHandler)
	router.POST("/pesanan/:kode/batal", orderCancelHandler)
	router.POST("/pesanan/:kode/simulasi-bayar", simulateQRISPaymentHandler)
	router.GET("/api/pesanan/:kode/status", orderStatusAPIHandler)
	router.POST("/webhook/midtrans", midtransWebhookHandler)

	router.GET("/lacak", trackPageHandler)
	router.POST("/lacak", trackSearchHandler)

	// Customer Auth & Account
	router.GET("/masuk", loginPageHandler)
	router.POST("/masuk", loginPostHandler)
	router.GET("/daftar", registerPageHandler)
	router.POST("/daftar", registerPostHandler)
	router.GET("/keluar", logoutHandler)
	router.POST("/keluar", logoutHandler)
	router.GET("/akun", accountPageHandler)

	// Portal Autentikasi Pegawai (Kasir & Admin)
	router.GET("/pegawai/masuk", staffLoginPageHandler)
	router.POST("/pegawai/masuk", staffLoginPostHandler)
	router.GET("/pegawai/keluar", staffLogoutHandler)
	router.POST("/pegawai/keluar", staffLogoutHandler)

	// Panel Kasir (Dedicated Cashier Page)
	router.GET("/kasir", cashierPageHandler)
	router.GET("/kasir/pesanan/:kode/cetak", cashierPrintReceiptHandler)
	router.POST("/kasir/pesanan/:kode/status", cashierUpdateOrderStatusHandler)

	// Dashboard Admin Lengkap (Dedicated Store Management Page - Admin Only)
	router.GET("/admin", adminDashboardHandler)
	router.GET("/admin/laporan/ekspor", adminExportSalesReportCSVHandler)
	router.POST("/admin/menu/tambah", adminAddProductHandler)
	router.POST("/admin/menu/harga", adminUpdatePriceHandler)
	router.POST("/admin/stok/sesuaikan", adminAdjustStockHandler)
	router.POST("/admin/pengaturan", adminUpdateSettingsHandler)

	// Kompatibilitas rute lama admin/pesanan
	router.GET("/admin/pesanan", adminOrdersPageHandler)
	router.POST("/admin/pesanan/:kode/status", adminUpdateOrderStatusHandler)

	return router
}
