package customer

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestRegisterAndAuthCustomer(t *testing.T) {
	store := NewStore()

	// 1. Pelanggan Mahasiswa sukses mendaftar dengan NIM
	mhs, err := store.Register(RegisterInput{
		Type:     CustomerTypeMahasiswa,
		Name:     "Ahmad Dahlan",
		Phone:    "081122334455",
		Email:    "ahmad@mhs.unism.ac.id",
		NIM:      "221001099",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("gagal daftar mahasiswa: %v", err)
	}
	if mhs.Type != CustomerTypeMahasiswa || mhs.NIM != "221001099" {
		t.Errorf("data mahasiswa tidak sesuai: %+v", mhs)
	}

	// 2. Mahasiswa tanpa NIM harus gagal
	_, err = store.Register(RegisterInput{
		Type:     CustomerTypeMahasiswa,
		Name:     "Tanpa NIM",
		Phone:    "081122334456",
		Email:    "tanpa@mhs.unism.ac.id",
		NIM:      "",
		Password: "password123",
	})
	if err != ErrNIMRequired {
		t.Errorf("diharapkan ErrNIMRequired, dapat: %v", err)
	}

	// 3. Pelanggan Umum sukses mendaftar tanpa NIM
	umum, err := store.Register(RegisterInput{
		Type:     CustomerTypeUmum,
		Name:     "Rina Wati",
		Phone:    "085566778899",
		Email:    "rina@gmail.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("gagal daftar umum: %v", err)
	}
	if umum.Type != CustomerTypeUmum || umum.NIM != "" {
		t.Errorf("data umum tidak sesuai: %+v", umum)
	}

	// 4. Login Mahasiswa bisa pakai NIM
	authMhsByNIM, err := store.Authenticate("221001099", "password123")
	if err != nil {
		t.Fatalf("gagal auth via NIM: %v", err)
	}
	if authMhsByNIM.ID != mhs.ID {
		t.Errorf("ID mahasiswa tidak cocok")
	}

	// 5. Login Umum pakai Telepon
	authUmum, err := store.Authenticate("085566778899", "password123")
	if err != nil {
		t.Fatalf("gagal auth via Phone: %v", err)
	}
	if authUmum.ID != umum.ID {
		t.Errorf("ID umum tidak cocok")
	}

	// 6. Sandi salah harus gagal
	_, err = store.Authenticate("085566778899", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("diharapkan ErrInvalidCredentials, dapat: %v", err)
	}

	// 7. Login menggunakan Email harus ditolak (hanya nomor telepon atau NIM yang diizinkan)
	_, err = store.Authenticate("ahmad@mhs.unism.ac.id", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("login via email harus ditolak dengan ErrInvalidCredentials, dapat: %v", err)
	}
	_, err = store.Authenticate("rina@gmail.com", "password123")
	if err != ErrInvalidCredentials {
		t.Errorf("login via email umum harus ditolak dengan ErrInvalidCredentials, dapat: %v", err)
	}
}

func TestImportCustomersPreservesHashAndRestoresSequence(t *testing.T) {
	store := NewStore()

	hash, err := bcrypt.GenerateFromPassword([]byte("sandi123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("gagal membuat hash: %v", err)
	}

	store.ImportCustomers([]*Customer{{
		ID:           "CUST-0009",
		Type:         CustomerTypeUmum,
		Name:         "Akun Lama",
		Phone:        "081300000009",
		Email:        "lama@example.com",
		PasswordHash: string(hash),
	}})

	// Registrasi berikutnya tidak boleh memakai ID yang sudah ada.
	baru, err := store.Register(RegisterInput{
		Type:     CustomerTypeUmum,
		Name:     "Akun Baru",
		Phone:    "081300000010",
		Password: "sandi123",
	})
	if err != nil {
		t.Fatalf("gagal registrasi: %v", err)
	}
	if baru.ID != "CUST-0010" {
		t.Errorf("ID registrasi baru = %s, want CUST-0010", baru.ID)
	}

	// Impor data lama tanpa hash tidak boleh menghapus hash yang valid.
	store.ImportCustomers([]*Customer{{
		ID:    "CUST-0009",
		Type:  CustomerTypeUmum,
		Name:  "Akun Lama",
		Phone: "081300000009",
	}})
	if _, err := store.Authenticate("081300000009", "sandi123"); err != nil {
		t.Errorf("hash sandi akun lama seharusnya dipertahankan, dapat: %v", err)
	}
}

func TestCalculateFirstDiscount(t *testing.T) {
	// Mahasiswa baru (belum pakai diskon)
	mhs := &Customer{
		Type:              CustomerTypeMahasiswa,
		NIM:               "221001099",
		FirstDiscountUsed: false,
	}

	// Kasus 1: Subtotal Rp 20.000 (< 30.000) -> Diskon 25% = Rp 5.000
	disc, _ := mhs.CalculateFirstDiscount(20000)
	if disc != 5000 {
		t.Errorf("subtotal 20000: diskon diharapkan 5000, dapat %d", disc)
	}

	// Kasus 2: Subtotal Rp 30.000 (tepat 30.000) -> Diskon 25% = Rp 7.500
	disc, _ = mhs.CalculateFirstDiscount(30000)
	if disc != 7500 {
		t.Errorf("subtotal 30000: diskon diharapkan 7500, dapat %d", disc)
	}

	// Kasus 3: Subtotal Rp 60.000 (> 30.000) -> Maksimal pembelian 30.000, diskon 25% dari 30.000 = Rp 7.500
	disc, _ = mhs.CalculateFirstDiscount(60000)
	if disc != 7500 {
		t.Errorf("subtotal 60000: diskon diharapkan 7500 (max cap), dapat %d", disc)
	}

	// Kasus 4: Mahasiswa yang sudah pernah pakai diskon
	mhs.FirstDiscountUsed = true
	disc, _ = mhs.CalculateFirstDiscount(30000)
	if disc != 0 {
		t.Errorf("mahasiswa sudah pakai promo: diskon diharapkan 0, dapat %d", disc)
	}

	// Kasus 5: Pelanggan umum tidak mendapatkan diskon mahasiswa
	umum := &Customer{
		Type:              CustomerTypeUmum,
		FirstDiscountUsed: false,
	}
	disc, _ = umum.CalculateFirstDiscount(30000)
	if disc != 0 {
		t.Errorf("pelanggan umum: diskon diharapkan 0, dapat %d", disc)
	}
}
