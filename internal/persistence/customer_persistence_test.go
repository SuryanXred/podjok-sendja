package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/order"
)

// Regression test: kata sandi pelanggan harus ikut tersimpan saat data
// dipersistensikan ke berkas, supaya pengguna masih bisa login setelah
// server dimulai ulang tanpa MySQL.
func TestCustomerPasswordHashPersistedAndLoginWorksAfterReload(t *testing.T) {
	dir := t.TempDir()
	Init(dir)

	const phone = "081200000001"
	const password = "sandiuji123"

	cust, err := customer.DefaultStore.Register(customer.RegisterInput{
		Type:     customer.CustomerTypeMahasiswa,
		Name:     "Pelanggan Persistensi",
		Phone:    phone,
		Email:    "persistensi@example.com",
		NIM:      "999000001",
		Password: password,
	})
	if err != nil {
		t.Fatalf("registrasi gagal: %v", err)
	}

	if _, err := customer.DefaultStore.Authenticate(phone, password); err != nil {
		t.Fatalf("login sebelum reload seharusnya berhasil: %v", err)
	}

	SaveCustomers()

	// 1. Berkas harus memuat password_hash yang tidak kosong.
	raw, err := os.ReadFile(filepath.Join(dir, "customers.json"))
	if err != nil {
		t.Fatalf("gagal membaca customers.json: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("customers.json bukan JSON yang valid: %v", err)
	}

	var found bool
	for _, row := range rows {
		if row["id"] != cust.ID {
			continue
		}
		found = true
		hash, _ := row["password_hash"].(string)
		if hash == "" {
			t.Fatalf("customers.json tidak menyimpan password_hash untuk %s: %s", cust.ID, string(raw))
		}
	}
	if !found {
		t.Fatalf("pelanggan %s tidak ditemukan di customers.json", cust.ID)
	}

	// 2. Simulasikan server restart: muat ulang dari berkas, lalu login.
	Init(dir)
	if _, err := customer.DefaultStore.Authenticate(phone, password); err != nil {
		t.Fatalf("login setelah server dimulai ulang gagal: %v", err)
	}
}

func TestReconcileCustomerDiscountsFromExistingOrders(t *testing.T) {
	dir := t.TempDir()
	Init(dir)

	// Registrasi mahasiswa baru
	cust, err := customer.DefaultStore.Register(customer.RegisterInput{
		Type:     customer.CustomerTypeMahasiswa,
		Name:     "Mahasiswa Uji Diskon",
		Phone:    "081999111222",
		Email:    "ujidiskon@example.com",
		NIM:      "11203462410999",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("registrasi gagal: %v", err)
	}

	// Buat pesanan dengan diskon untuk customer ini
	testOrder := &order.Order{
		Code:         "PDS-TEST-RECON-01",
		CustomerID:   cust.ID,
		CustomerType: "mahasiswa",
		Subtotal:     30000,
		Discount:     7500,
		Total:        22500,
	}
	order.DefaultStore.ImportState(map[string]*order.Order{testOrder.Code: testOrder}, 1)

	// Pastikan sebelum rekonsiliasi jika FirstDiscountUsed false, setelah reconcile menjadi true
	cust.FirstDiscountUsed = false
	reconcileCustomerDiscounts()

	reloadedCust, err := customer.DefaultStore.GetByID(cust.ID)
	if err != nil {
		t.Fatalf("pelanggan tidak ditemukan: %v", err)
	}
	if !reloadedCust.FirstDiscountUsed {
		t.Errorf("expected FirstDiscountUsed to be true after reconciliation with discounted order")
	}
}
