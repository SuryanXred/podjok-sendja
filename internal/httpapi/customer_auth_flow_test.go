package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/order"
)

func TestCustomerAuthAndStudentDiscountFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	// ----------------------------------------------------
	// 1. Registrasi Mahasiswa (Universitas Sari Mulia dengan NIM)
	// ----------------------------------------------------
	mhsForm := url.Values{}
	mhsForm.Set("type", "mahasiswa")
	mhsForm.Set("nim", "231001050")
	mhsForm.Set("name", "Dewi Sartika")
	mhsForm.Set("phone", "081344556677")
	mhsForm.Set("email", "dewi.mhs@unism.ac.id")
	mhsForm.Set("password", "sandi123")

	reqMhsReg := httptest.NewRequest(http.MethodPost, "/daftar", strings.NewReader(mhsForm.Encode()))
	reqMhsReg.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respMhsReg := httptest.NewRecorder()
	router.ServeHTTP(respMhsReg, reqMhsReg)

	if respMhsReg.Code != http.StatusSeeOther {
		t.Fatalf("POST /daftar (mahasiswa) status = %d, want %d", respMhsReg.Code, http.StatusSeeOther)
	}

	sessionCookie := respMhsReg.Header().Get("Set-Cookie")
	if !strings.Contains(sessionCookie, sessionCookieName) {
		t.Fatalf("session cookie missing in registration response: %q", sessionCookie)
	}
	rawSessionCookie := strings.Split(strings.Split(sessionCookie, ";")[0], "=")[1]

	// ----------------------------------------------------
	// 2. Buka Halaman Akun Mahasiswa
	// ----------------------------------------------------
	reqAcc := httptest.NewRequest(http.MethodGet, "/akun", nil)
	reqAcc.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	respAcc := httptest.NewRecorder()
	router.ServeHTTP(respAcc, reqAcc)

	if respAcc.Code != http.StatusOK {
		t.Fatalf("GET /akun status = %d, want %d", respAcc.Code, http.StatusOK)
	}
	bodyAcc := respAcc.Body.String()
	if !strings.Contains(bodyAcc, "Dewi Sartika") || !strings.Contains(bodyAcc, "231001050") {
		t.Errorf("halaman akun tidak memuat nama atau NIM mahasiswa: %s", bodyAcc)
	}
	if !strings.Contains(bodyAcc, "Diskon 25% Pembelian Pertama Aktif") {
		t.Errorf("halaman akun mahasiswa baru harus menampilkan info promo aktif: %s", bodyAcc)
	}

	// ----------------------------------------------------
	// 3. Tambahkan Americano Ice (Rp 15.000) ke Keranjang
	// ----------------------------------------------------
	cartForm := url.Values{}
	cartForm.Set("product_slug", "americano")
	cartForm.Set("variant", "Ice")

	reqCartAdd := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(cartForm.Encode()))
	reqCartAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respCartAdd := httptest.NewRecorder()
	router.ServeHTTP(respCartAdd, reqCartAdd)

	cartCookie := respCartAdd.Header().Get("Set-Cookie")
	rawCartCookie := strings.Split(strings.Split(cartCookie, ";")[0], "=")[1]

	// ----------------------------------------------------
	// 4. Buka Keranjang saat login sebagai mahasiswa
	// ----------------------------------------------------
	reqCartView := httptest.NewRequest(http.MethodGet, "/keranjang", nil)
	reqCartView.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	reqCartView.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCartCookie})
	respCartView := httptest.NewRecorder()
	router.ServeHTTP(respCartView, reqCartView)

	if respCartView.Code != http.StatusOK {
		t.Fatalf("GET /keranjang status = %d, want %d", respCartView.Code, http.StatusOK)
	}
	bodyCartView := respCartView.Body.String()
	// Subtotal Rp 15.000 -> Diskon 25% = Rp 3.750 -> Total Rp 11.250
	if !strings.Contains(bodyCartView, "Diskon 25%") || !strings.Contains(bodyCartView, "11.250") {
		t.Errorf("keranjang mahasiswa baru harus menampilkan diskon 25%% (total Rp 11.250): %s", bodyCartView)
	}

	// ----------------------------------------------------
	// 5. Checkout Pesanan Pertama Mahasiswa
	// ----------------------------------------------------
	coForm := url.Values{}
	coForm.Set("customer_name", "Dewi Sartika")
	coForm.Set("customer_phone", "081344556677")
	coForm.Set("order_type", "dine_in")
	coForm.Set("table_number", "05")
	coForm.Set("payment_method", "cash")

	reqCO := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(coForm.Encode()))
	reqCO.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCO.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	reqCO.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCartCookie})
	respCO := httptest.NewRecorder()
	router.ServeHTTP(respCO, reqCO)

	if respCO.Code != http.StatusSeeOther {
		t.Fatalf("POST /checkout status = %d, want %d", respCO.Code, http.StatusSeeOther)
	}

	orderURL := respCO.Header().Get("Location")
	if !strings.HasPrefix(orderURL, "/pesanan/PDS-") {
		t.Fatalf("Location = %q, want /pesanan/PDS-...", orderURL)
	}
	orderCode := strings.TrimPrefix(orderURL, "/pesanan/")

	// Periksa objek pesanan di Store
	savedOrder, err := order.DefaultStore.GetOrderByCode(orderCode)
	if err != nil {
		t.Fatalf("pesanan tidak ditemukan di store: %v", err)
	}
	if savedOrder.CustomerType != "mahasiswa" || savedOrder.CustomerNIM != "231001050" {
		t.Errorf("tipe atau NIM pesanan tidak tersimpan dengan benar: %+v", savedOrder)
	}
	if savedOrder.Subtotal != 15000 {
		t.Errorf("subtotal = %d, want 15000", savedOrder.Subtotal)
	}
	if savedOrder.Discount != 3750 { // 25% of 15000
		t.Errorf("diskon = %d, want 3750", savedOrder.Discount)
	}
	if savedOrder.Total != 11250 {
		t.Errorf("total bayar = %d, want 11250", savedOrder.Total)
	}

	// ----------------------------------------------------
	// 6. Buka Nota Pesanan: Harus memuat NIM dan baris diskon
	// ----------------------------------------------------
	reqReceipt := httptest.NewRequest(http.MethodGet, "/pesanan/"+orderCode, nil)
	reqReceipt.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	respReceipt := httptest.NewRecorder()
	router.ServeHTTP(respReceipt, reqReceipt)

	if respReceipt.Code != http.StatusOK {
		t.Fatalf("GET /pesanan/%s status = %d, want %d", orderCode, respReceipt.Code, http.StatusOK)
	}
	bodyReceipt := respReceipt.Body.String()
	if !strings.Contains(bodyReceipt, "231001050") || !strings.Contains(bodyReceipt, "3.750") {
		t.Errorf("nota pesanan tidak menampilkan NIM atau nominal diskon: %s", bodyReceipt)
	}

	// ----------------------------------------------------
	// 7. Pembelian Kedua oleh Mahasiswa yang Sama: Diskon harus 0
	// ----------------------------------------------------
	// Tambah lagi Americano Ice ke keranjang
	reqCartAdd2 := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(cartForm.Encode()))
	reqCartAdd2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respCartAdd2 := httptest.NewRecorder()
	router.ServeHTTP(respCartAdd2, reqCartAdd2)

	cartCookie2 := respCartAdd2.Header().Get("Set-Cookie")
	rawCartCookie2 := strings.Split(strings.Split(cartCookie2, ";")[0], "=")[1]

	// Checkout pesanan kedua
	reqCO2 := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(coForm.Encode()))
	reqCO2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCO2.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	reqCO2.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCartCookie2})
	respCO2 := httptest.NewRecorder()
	router.ServeHTTP(respCO2, reqCO2)

	if respCO2.Code != http.StatusSeeOther {
		t.Fatalf("checkout kedua status = %d, want %d", respCO2.Code, http.StatusSeeOther)
	}
	orderCode2 := strings.TrimPrefix(respCO2.Header().Get("Location"), "/pesanan/")
	savedOrder2, err := order.DefaultStore.GetOrderByCode(orderCode2)
	if err != nil {
		t.Fatalf("pesanan kedua tidak ditemukan: %v", err)
	}
	if savedOrder2.Discount != 0 {
		t.Errorf("diskon pada pembelian kedua harus 0, dapat: %d", savedOrder2.Discount)
	}
	if savedOrder2.Total != 15000 {
		t.Errorf("total pesanan kedua harus normal Rp 15.000, dapat: %d", savedOrder2.Total)
	}
}

func TestMaxPurchaseDiscountCap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	// Daftar mahasiswa baru
	mhsForm := url.Values{}
	mhsForm.Set("type", "mahasiswa")
	mhsForm.Set("nim", "231001099")
	mhsForm.Set("name", "Ilham Prasetyo")
	mhsForm.Set("phone", "081999888777")
	mhsForm.Set("password", "sandi123")

	reqReg := httptest.NewRequest(http.MethodPost, "/daftar", strings.NewReader(mhsForm.Encode()))
	reqReg.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respReg := httptest.NewRecorder()
	router.ServeHTTP(respReg, reqReg)

	sessionCookie := respReg.Header().Get("Set-Cookie")
	rawSessionCookie := strings.Split(strings.Split(sessionCookie, ";")[0], "=")[1]

	// Tambahkan item Mix Platter (Rp 25.000) sebanyak 2 porsi -> Subtotal Rp 50.000 (> Rp 30.000)
	cartForm := url.Values{}
	cartForm.Set("product_slug", "mix-platter")

	reqAdd := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(cartForm.Encode()))
	reqAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respAdd := httptest.NewRecorder()
	router.ServeHTTP(respAdd, reqAdd)

	rawCartCookie := strings.Split(strings.Split(respAdd.Header().Get("Set-Cookie"), ";")[0], "=")[1]

	// Tambah 1 porsi lagi lewat update delta
	upForm := url.Values{}
	upForm.Set("index", "0")
	upForm.Set("delta", "1")
	reqUp := httptest.NewRequest(http.MethodPost, "/cart/update", strings.NewReader(upForm.Encode()))
	reqUp.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUp.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCartCookie})
	respUp := httptest.NewRecorder()
	router.ServeHTTP(respUp, reqUp)
	rawCartCookie = strings.Split(strings.Split(respUp.Header().Get("Set-Cookie"), ";")[0], "=")[1]

	// Checkout pesanan dengan subtotal Rp 50.000
	coForm := url.Values{}
	coForm.Set("customer_name", "Ilham Prasetyo")
	coForm.Set("customer_phone", "081999888777")
	coForm.Set("order_type", "takeaway")
	coForm.Set("payment_method", "cash")

	reqCO := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(coForm.Encode()))
	reqCO.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCO.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	reqCO.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCartCookie})
	respCO := httptest.NewRecorder()
	router.ServeHTTP(respCO, reqCO)

	orderCode := strings.TrimPrefix(respCO.Header().Get("Location"), "/pesanan/")
	savedOrder, err := order.DefaultStore.GetOrderByCode(orderCode)
	if err != nil {
		t.Fatalf("pesanan tidak ditemukan: %v", err)
	}

	if savedOrder.Subtotal != 40000 {
		t.Errorf("subtotal = %d, want 40000", savedOrder.Subtotal)
	}
	// Pembelian maksimal 30.000 yang dihitung diskon 25% -> diskon maksimal = 25% x 30.000 = Rp 7.500
	if savedOrder.Discount != 7500 {
		t.Errorf("diskon cap = %d, want 7500 (25%% dari max 30.000)", savedOrder.Discount)
	}
	// Total = 40.000 - 7.500 = 32.500
	if savedOrder.Total != 32500 {
		t.Errorf("total bayar = %d, want 32500", savedOrder.Total)
	}
}

func TestPelangganUmumLoginAndOrderFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	// 1. Registrasi Pelanggan Umum
	regForm := url.Values{}
	regForm.Set("type", "umum")
	regForm.Set("name", "Bapak Suryono")
	regForm.Set("phone", "082199887766")
	regForm.Set("email", "suryono@gmail.com")
	regForm.Set("password", "sandi123")

	reqReg := httptest.NewRequest(http.MethodPost, "/daftar", strings.NewReader(regForm.Encode()))
	reqReg.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respReg := httptest.NewRecorder()
	router.ServeHTTP(respReg, reqReg)

	if respReg.Code != http.StatusSeeOther {
		t.Fatalf("daftar umum status = %d, want %d", respReg.Code, http.StatusSeeOther)
	}

	// 2a. Percobaan login menggunakan Email ditolak (422 Unprocessable Entity)
	emailLoginForm := url.Values{}
	emailLoginForm.Set("identifier", "suryono@gmail.com")
	emailLoginForm.Set("password", "sandi123")
	reqEmailLogin := httptest.NewRequest(http.MethodPost, "/masuk", strings.NewReader(emailLoginForm.Encode()))
	reqEmailLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respEmailLogin := httptest.NewRecorder()
	router.ServeHTTP(respEmailLogin, reqEmailLogin)

	if respEmailLogin.Code != http.StatusUnprocessableEntity {
		t.Fatalf("login via email harus ditolak dengan status 422, dapat %d", respEmailLogin.Code)
	}

	// 2. Login Pelanggan Umum via Phone
	loginForm := url.Values{}
	loginForm.Set("identifier", "082199887766")
	loginForm.Set("password", "sandi123")

	reqLogin := httptest.NewRequest(http.MethodPost, "/masuk", strings.NewReader(loginForm.Encode()))
	reqLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respLogin := httptest.NewRecorder()
	router.ServeHTTP(respLogin, reqLogin)

	if respLogin.Code != http.StatusSeeOther {
		t.Fatalf("login umum status = %d, want %d", respLogin.Code, http.StatusSeeOther)
	}
	sessionCookie := respLogin.Header().Get("Set-Cookie")
	rawSessionCookie := strings.Split(strings.Split(sessionCookie, ";")[0], "=")[1]

	// 3. Tambahkan menu dan checkout
	cartForm := url.Values{}
	cartForm.Set("product_slug", "espresso")

	reqAdd := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(cartForm.Encode()))
	reqAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respAdd := httptest.NewRecorder()
	router.ServeHTTP(respAdd, reqAdd)
	rawCartCookie := strings.Split(strings.Split(respAdd.Header().Get("Set-Cookie"), ";")[0], "=")[1]

	coForm := url.Values{}
	coForm.Set("customer_name", "Bapak Suryono")
	coForm.Set("customer_phone", "082199887766")
	coForm.Set("order_type", "dine_in")
	coForm.Set("table_number", "01")
	coForm.Set("payment_method", "cash")

	reqCO := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(coForm.Encode()))
	reqCO.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCO.AddCookie(&http.Cookie{Name: sessionCookieName, Value: rawSessionCookie})
	reqCO.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCartCookie})
	respCO := httptest.NewRecorder()
	router.ServeHTTP(respCO, reqCO)

	orderCode := strings.TrimPrefix(respCO.Header().Get("Location"), "/pesanan/")
	savedOrder, err := order.DefaultStore.GetOrderByCode(orderCode)
	if err != nil {
		t.Fatalf("pesanan tidak ditemukan: %v", err)
	}

	// Pelanggan umum tidak memiliki diskon mahasiswa
	if savedOrder.CustomerType != "umum" {
		t.Errorf("customer_type = %s, want umum", savedOrder.CustomerType)
	}
	if savedOrder.CustomerNIM != "" {
		t.Errorf("customer_nim umum harus kosong, dapat: %s", savedOrder.CustomerNIM)
	}
	if savedOrder.Discount != 0 {
		t.Errorf("diskon pelanggan umum harus 0, dapat: %d", savedOrder.Discount)
	}
	if savedOrder.Total != 10000 {
		t.Errorf("total bayar umum harus 10000, dapat: %d", savedOrder.Total)
	}
}
