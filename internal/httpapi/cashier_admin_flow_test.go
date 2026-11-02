package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/staff"
)

func loginStaffHelper(t *testing.T, router *gin.Engine, identifier, password string) *http.Cookie {
	t.Helper()
	form := url.Values{}
	form.Set("identifier", identifier)
	form.Set("password", password)

	req, _ := http.NewRequest(http.MethodPost, "/pegawai/masuk", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect on staff login for %s, got %d", identifier, w.Code)
	}

	for _, c := range w.Result().Cookies() {
		if c.Name == staffSessionCookieName {
			return c
		}
	}

	t.Fatalf("session cookie %s not found in login response", staffSessionCookieName)
	return nil
}

func TestUnauthenticatedStaffRedirectsToLogin(t *testing.T) {
	router := NewRouter()

	// 1. Tanpa login akses /kasir harus dialihkan ke /pegawai/masuk
	reqKasir, _ := http.NewRequest(http.MethodGet, "/kasir", nil)
	wKasir := httptest.NewRecorder()
	router.ServeHTTP(wKasir, reqKasir)

	if wKasir.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect for unauthenticated /kasir, got %d", wKasir.Code)
	}
	if loc := wKasir.Header().Get("Location"); loc != "/pegawai/masuk?redirect=/kasir" {
		t.Errorf("expected redirect location '/pegawai/masuk?redirect=/kasir', got %q", loc)
	}

	// 2. Tanpa login akses /admin harus dialihkan ke /pegawai/masuk
	reqAdmin, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusSeeOther {
		t.Errorf("expected 303 redirect for unauthenticated /admin, got %d", wAdmin.Code)
	}
	if loc := wAdmin.Header().Get("Location"); loc != "/pegawai/masuk?redirect=/admin" {
		t.Errorf("expected redirect location '/pegawai/masuk?redirect=/admin', got %q", loc)
	}
}

func TestStaffLoginPageLoads(t *testing.T) {
	router := NewRouter()

	req, _ := http.NewRequest(http.MethodGet, "/pegawai/masuk", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /pegawai/masuk, got %d", w.Code)
	}

	body := w.Body.String()
	wantStrings := []string{
		"Masuk Portal Pegawai",
		"Username Pegawai",
		"Kata Sandi",
		"Kredensial Akun Percobaan",
		"admin123",
		"kasir123",
	}

	for _, s := range wantStrings {
		if !strings.Contains(body, s) {
			t.Errorf("staff login page missing expected string: %q", s)
		}
	}
}

func TestCashierRoleAccessAndForbiddenAdmin(t *testing.T) {
	router := NewRouter()

	// 1. Login sebagai Kasir
	cashierCookie := loginStaffHelper(t, router, "kasir", "kasir123")

	// 2. Akses ke /kasir harus berhasil (200 OK)
	reqKasir, _ := http.NewRequest(http.MethodGet, "/kasir", nil)
	reqKasir.AddCookie(cashierCookie)
	wKasir := httptest.NewRecorder()
	router.ServeHTTP(wKasir, reqKasir)

	if wKasir.Code != http.StatusOK {
		t.Fatalf("cashier should have access to /kasir, got status %d", wKasir.Code)
	}
	bodyKasir := wKasir.Body.String()
	if !strings.Contains(bodyKasir, "Panel Kasir &amp; Antrean") {
		t.Error("cashier page title missing")
	}
	if !strings.Contains(bodyKasir, "Kasir Meja Depan") {
		t.Error("cashier staff name missing from page")
	}

	// 3. Kasir mencoba akses ke /admin: HARUS DITOLAK (403 Forbidden)!
	reqAdmin, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	reqAdmin.AddCookie(cashierCookie)
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusForbidden {
		t.Fatalf("cashier accessing /admin MUST be forbidden (403), got status %d", wAdmin.Code)
	}
	bodyForbidden := wAdmin.Body.String()
	if !strings.Contains(bodyForbidden, "Akses Ditolak (403)") {
		t.Error("expected 403 Forbidden page title for cashier accessing admin dashboard")
	}
	if !strings.Contains(bodyForbidden, "Hak Akses Tidak Mencukupi") {
		t.Error("expected access warning explanation on forbidden page")
	}
}

func TestAdminRoleFullAccess(t *testing.T) {
	router := NewRouter()

	// 1. Login sebagai Admin
	adminCookie := loginStaffHelper(t, router, "admin", "admin123")

	// 2. Admin dapat mengakses /admin (200 OK)
	reqAdmin, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	reqAdmin.AddCookie(adminCookie)
	wAdmin := httptest.NewRecorder()
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("admin should have access to /admin, got status %d", wAdmin.Code)
	}
	if !strings.Contains(wAdmin.Body.String(), "Dashboard Manajemen") {
		t.Error("admin dashboard title missing")
	}
	if !strings.Contains(wAdmin.Body.String(), "Manajer Kafe") {
		t.Error("admin staff name missing from page")
	}

	// 3. Admin juga memiliki hak akses penuh ke /kasir (200 OK)
	reqKasir, _ := http.NewRequest(http.MethodGet, "/kasir", nil)
	reqKasir.AddCookie(adminCookie)
	wKasir := httptest.NewRecorder()
	router.ServeHTTP(wKasir, reqKasir)

	if wKasir.Code != http.StatusOK {
		t.Fatalf("admin should also have access to /kasir, got status %d", wKasir.Code)
	}
}

func TestCashierOrderLifecycleWithAuth(t *testing.T) {
	router := NewRouter()
	cashierCookie := loginStaffHelper(t, router, "kasir", "kasir123")

	// 1. Buat pesanan baru
	createdOrder, err := order.DefaultStore.CreateOrder(order.CreateOrderInput{
		CustomerName:  "Dina Kasir Test",
		CustomerPhone: "08123456789",
		CustomerType:  "umum",
		OrderType:     order.OrderTypeDineIn,
		TableNumber:   "4",
		PaymentMethod: order.PaymentMethodCash,
		Cart: order.Cart{
			TotalQuantity: 2,
			Subtotal:      24000,
			Items: []order.CartItem{
				{
					ProductSlug: "americano",
					ProductName: "Americano",
					Variant:     "Hot",
					UnitPrice:   12000,
					Quantity:    2,
					Subtotal:    24000,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create test order: %v", err)
	}

	// 2. Periksa pesanan muncul di halaman kasir
	req, _ := http.NewRequest(http.MethodGet, "/kasir", nil)
	req.AddCookie(cashierCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), createdOrder.Code) {
		t.Errorf("order %s not found on /kasir page", createdOrder.Code)
	}

	// 3. Kasir verifikasi pembayaran
	formPay := url.Values{}
	formPay.Set("status", string(order.OrderStatusPaid))
	formPay.Set("payment_status", string(order.PaymentStatusPaid))
	reqPay, _ := http.NewRequest(http.MethodPost, "/kasir/pesanan/"+createdOrder.Code+"/status", strings.NewReader(formPay.Encode()))
	reqPay.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPay.AddCookie(cashierCookie)
	wPay := httptest.NewRecorder()
	router.ServeHTTP(wPay, reqPay)

	if wPay.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after payment verification, got %d", wPay.Code)
	}

	updated, err := order.DefaultStore.GetOrderByCode(createdOrder.Code)
	if err != nil || updated.OrderStatus != order.OrderStatusPaid {
		t.Errorf("expected order status to be 'paid', got %v", updated.OrderStatus)
	}

	// 4. Kasir tandai siap diambil
	formReady := url.Values{}
	formReady.Set("status", string(order.OrderStatusReady))
	reqReady, _ := http.NewRequest(http.MethodPost, "/kasir/pesanan/"+createdOrder.Code+"/status", strings.NewReader(formReady.Encode()))
	reqReady.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqReady.AddCookie(cashierCookie)
	wReady := httptest.NewRecorder()
	router.ServeHTTP(wReady, reqReady)

	updated, _ = order.DefaultStore.GetOrderByCode(createdOrder.Code)
	if updated.OrderStatus != order.OrderStatusReady {
		t.Errorf("expected order status to be 'ready', got %v", updated.OrderStatus)
	}
}

func TestAdminManagementActionsFlow(t *testing.T) {
	router := NewRouter()
	adminCookie := loginStaffHelper(t, router, "admin", "admin123")

	// 1. Tambah menu baru oleh admin
	formAdd := url.Values{}
	formAdd.Set("category_slug", "coffee")
	formAdd.Set("name", "Kopi Rempah Sari Mulia")
	formAdd.Set("base_price", "17000")
	formAdd.Set("variants", "Ice:3000, Hot:0")

	reqAdd, _ := http.NewRequest(http.MethodPost, "/admin/menu/tambah", strings.NewReader(formAdd.Encode()))
	reqAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAdd.AddCookie(adminCookie)
	wAdd := httptest.NewRecorder()
	router.ServeHTTP(wAdd, reqAdd)

	if wAdd.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after add product, got %d", wAdd.Code)
	}

	// 2. Sesuaikan stok menu oleh admin
	formStock := url.Values{}
	formStock.Set("product_slug", "kopi-rempah-sari-mulia")
	formStock.Set("action_type", "add")
	formStock.Set("quantity", "25")
	formStock.Set("reason", "Pasokan bahan rempah datang")

	reqStock, _ := http.NewRequest(http.MethodPost, "/admin/stok/sesuaikan", strings.NewReader(formStock.Encode()))
	reqStock.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqStock.AddCookie(adminCookie)
	wStock := httptest.NewRecorder()
	router.ServeHTTP(wStock, reqStock)

	if wStock.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after stock adjust, got %d", wStock.Code)
	}

	// 3. Update pengaturan kafe oleh admin
	formSettings := url.Values{}
	formSettings.Set("is_open", "true")
	formSettings.Set("opening_hours", "08.00 - 22.00 WITA")
	formSettings.Set("contact_phone", "0812-1111-2222")
	formSettings.Set("table_count", "18")
	formSettings.Set("announcement", "Pengumuman kafe diperbarui oleh admin.")

	reqSettings, _ := http.NewRequest(http.MethodPost, "/admin/pengaturan", strings.NewReader(formSettings.Encode()))
	reqSettings.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqSettings.AddCookie(adminCookie)
	wSettings := httptest.NewRecorder()
	router.ServeHTTP(wSettings, reqSettings)

	if wSettings.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after settings update, got %d", wSettings.Code)
	}

	// 4. Logout staff
	reqLogout, _ := http.NewRequest(http.MethodGet, "/pegawai/keluar", nil)
	reqLogout.AddCookie(adminCookie)
	wLogout := httptest.NewRecorder()
	router.ServeHTTP(wLogout, reqLogout)

	if wLogout.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect on logout, got %d", wLogout.Code)
	}
}

func TestStaffStoreAuthenticationErrors(t *testing.T) {
	_, err := staff.DefaultStore.Authenticate("nonexistent", "wrongpass")
	if err != staff.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	_, err = staff.DefaultStore.Authenticate("admin", "wrongpassword")
	if err != staff.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got %v", err)
	}

	// Login pegawai menggunakan email ditolak (hanya username)
	_, err = staff.DefaultStore.Authenticate("admin@sendja.com", "admin123")
	if err != staff.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials when staff attempts login with email, got %v", err)
	}
	_, err = staff.DefaultStore.Authenticate("kasir@sendja.com", "kasir123")
	if err != staff.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials when cashier attempts login with email, got %v", err)
	}
}
