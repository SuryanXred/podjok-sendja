package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caffe-podjok-sendja/internal/order"
)

func TestAdminExportSalesReportCSV(t *testing.T) {
	router := NewRouter()

	// 1. Tanpa autentikasi, redirect ke login pegawai
	reqUnauth, _ := http.NewRequest(http.MethodGet, "/admin/laporan/ekspor", nil)
	wUnauth := httptest.NewRecorder()
	router.ServeHTTP(wUnauth, reqUnauth)

	if wUnauth.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for unauthenticated export, got %d", wUnauth.Code)
	}

	// 2. Kasir biasa tidak boleh mengakses ekspor laporan admin
	cashierCookie := loginStaffHelper(t, router, "kasir", "kasir123")
	reqCashier, _ := http.NewRequest(http.MethodGet, "/admin/laporan/ekspor", nil)
	reqCashier.AddCookie(cashierCookie)
	wCashier := httptest.NewRecorder()
	router.ServeHTTP(wCashier, reqCashier)

	if wCashier.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for cashier accessing admin export, got %d", wCashier.Code)
	}

	// Buat pesanan contoh di database in-memory
	testInput := order.CreateOrderInput{
		CustomerName:  "Budi Santoso",
		CustomerPhone: "081234567899",
		CustomerType:  "mahasiswa",
		CustomerNIM:   "22010199",
		OrderType:     order.OrderTypeDineIn,
		TableNumber:   "5",
		PaymentMethod: order.PaymentMethodCash,
		Discount:      3000,
		DiscountNote:  "Diskon Mahasiswa Baru 25%",
		Cart: order.Cart{
			Items: []order.CartItem{
				{
					ProductSlug: "americano",
					ProductName: "Americano",
					Variant:     "Ice",
					UnitPrice:   15000,
					Quantity:    2,
					Subtotal:    30000,
				},
			},
			TotalQuantity: 2,
			Subtotal:      30000,
		},
	}
	sampleOrder, err := order.DefaultStore.CreateOrder(testInput)
	if err != nil {
		t.Fatalf("failed to create sample order: %v", err)
	}
	_ = order.DefaultStore.UpdateOrderStatus(sampleOrder.Code, order.OrderStatusCompleted, order.PaymentStatusPaid)

	// 3. Admin berhasil mengunduh CSV
	adminCookie := loginStaffHelper(t, router, "admin", "admin123")
	reqAdmin, _ := http.NewRequest(http.MethodGet, "/admin/laporan/ekspor?period=today", nil)
	reqAdmin.AddCookie(adminCookie)
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin export, got %d", wAdmin.Code)
	}

	contentType := wAdmin.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("expected Content-Type containing 'text/csv', got %q", contentType)
	}

	disposition := wAdmin.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment; filename=\"laporan-penjualan-caffe-podjok-sendja-today-") {
		t.Errorf("expected Content-Disposition attachment with filename, got %q", disposition)
	}

	bodyBytes := wAdmin.Body.Bytes()
	// Periksa UTF-8 BOM
	if !bytes.HasPrefix(bodyBytes, []byte("\xef\xbb\xbf")) {
		t.Errorf("expected UTF-8 BOM at start of CSV file for Excel compatibility")
	}

	bodyStr := string(bodyBytes)
	expectedHeaders := []string{
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

	for _, h := range expectedHeaders {
		if !strings.Contains(bodyStr, h) {
			t.Errorf("expected CSV header to contain %q", h)
		}
	}

	if !strings.Contains(bodyStr, sampleOrder.Code) {
		t.Errorf("expected CSV body to contain order code %q", sampleOrder.Code)
	}
	if !strings.Contains(bodyStr, "Budi Santoso") {
		t.Errorf("expected CSV body to contain customer name 'Budi Santoso'")
	}
	if !strings.Contains(bodyStr, "Americano (Ice) x2") {
		t.Errorf("expected CSV body to contain menu item details 'Americano (Ice) x2'")
	}
}

func TestCashierPrintReceipt(t *testing.T) {
	router := NewRouter()

	// 1. Tanpa autentikasi, akses cetak struk kasir dialihkan ke login
	reqUnauth, _ := http.NewRequest(http.MethodGet, "/kasir/pesanan/PDS-999999-0001/cetak", nil)
	wUnauth := httptest.NewRecorder()
	router.ServeHTTP(wUnauth, reqUnauth)

	if wUnauth.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for unauthenticated cashier receipt, got %d", wUnauth.Code)
	}

	// 2. Buat pesanan contoh
	testInput := order.CreateOrderInput{
		CustomerName:  "Siti Nurhaliza",
		CustomerPhone: "081298765432",
		CustomerType:  "umum",
		OrderType:     order.OrderTypeTakeaway,
		PaymentMethod: order.PaymentMethodQRIS,
		Cart: order.Cart{
			Items: []order.CartItem{
				{
					ProductSlug: "matcha-latte",
					ProductName: "Matcha Latte",
					UnitPrice:   18000,
					Quantity:    1,
					Subtotal:    18000,
				},
			},
			TotalQuantity: 1,
			Subtotal:      18000,
		},
	}
	sampleOrder, err := order.DefaultStore.CreateOrder(testInput)
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}
	_ = order.DefaultStore.UpdateOrderStatus(sampleOrder.Code, order.OrderStatusPaid, order.PaymentStatusPaid)

	// 3. Kasir yang sudah login dapat membuka halaman struk cetak
	cashierCookie := loginStaffHelper(t, router, "kasir", "kasir123")
	reqCashier, _ := http.NewRequest(http.MethodGet, "/kasir/pesanan/"+sampleOrder.Code+"/cetak", nil)
	reqCashier.AddCookie(cashierCookie)
	wCashier := httptest.NewRecorder()
	router.ServeHTTP(wCashier, reqCashier)

	if wCashier.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cashier print receipt, got %d", wCashier.Code)
	}

	body := wCashier.Body.String()
	if !strings.Contains(body, "Caffe Podjok Sendja") {
		t.Errorf("expected receipt to contain cafe brand name")
	}
	if !strings.Contains(body, "Universitas Sari Mulia") {
		t.Errorf("expected receipt to contain university name")
	}
	if !strings.Contains(body, sampleOrder.Code) {
		t.Errorf("expected receipt to contain order code %q", sampleOrder.Code)
	}
	if !strings.Contains(body, "Siti Nurhaliza") {
		t.Errorf("expected receipt to contain customer name")
	}
	if !strings.Contains(body, "Matcha Latte") {
		t.Errorf("expected receipt to contain item name 'Matcha Latte'")
	}
	if !strings.Contains(body, "Rp 18.000") {
		t.Errorf("expected receipt to contain formatted price 'Rp 18.000'")
	}
	if !strings.Contains(body, "*** LUNAS ***") {
		t.Errorf("expected receipt to show *** LUNAS *** badge")
	}
	if !strings.Contains(body, "window.print()") {
		t.Errorf("expected receipt to contain print trigger script")
	}
}

func TestPublicPrintReceipt(t *testing.T) {
	router := NewRouter()

	testInput := order.CreateOrderInput{
		CustomerName:  "Rina Wati",
		CustomerPhone: "081211112222",
		CustomerType:  "mahasiswa",
		CustomerNIM:   "23010555",
		OrderType:     order.OrderTypeDineIn,
		TableNumber:   "2",
		PaymentMethod: order.PaymentMethodCash,
		Cart: order.Cart{
			Items: []order.CartItem{
				{
					ProductSlug: "french-fries",
					ProductName: "French Fries",
					UnitPrice:   12000,
					Quantity:    1,
					Subtotal:    12000,
				},
			},
			TotalQuantity: 1,
			Subtotal:      12000,
		},
	}
	sampleOrder, _ := order.DefaultStore.CreateOrder(testInput)

	// Akses publik lewat /pesanan/:kode/cetak
	req, _ := http.NewRequest(http.MethodGet, "/pesanan/"+sampleOrder.Code+"/cetak", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for public print receipt, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, sampleOrder.Code) {
		t.Errorf("expected receipt to contain order code %q", sampleOrder.Code)
	}
	if !strings.Contains(body, "French Fries") {
		t.Errorf("expected receipt to contain 'French Fries'")
	}

	// 404 untuk kode tidak valid
	req404, _ := http.NewRequest(http.MethodGet, "/pesanan/KODE-TIDAK-ADA/cetak", nil)
	w404 := httptest.NewRecorder()
	router.ServeHTTP(w404, req404)

	if w404.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for non-existent order, got %d", w404.Code)
	}
}
