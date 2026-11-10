package customer

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCustomerNotFound   = errors.New("akun pelanggan tidak ditemukan")
	ErrInvalidCredentials = errors.New("nomor telepon, NIM, atau kata sandi tidak cocok")
	ErrNameRequired       = errors.New("nama lengkap wajib diisi minimal 2 karakter")
	ErrPhoneRequired      = errors.New("nomor WhatsApp atau telepon wajib diisi minimal 6 digit")
	ErrPasswordTooShort   = errors.New("kata sandi minimal 6 karakter")
	ErrInvalidType        = errors.New("tipe pelanggan harus Umum atau Mahasiswa")
	ErrNIMRequired        = errors.New("NIM wajib diisi untuk pelanggan mahasiswa")
	ErrPhoneAlreadyExists = errors.New("nomor telepon ini sudah terdaftar")
	ErrNIMAlreadyExists   = errors.New("NIM ini sudah terdaftar")
	ErrEmailAlreadyExists = errors.New("alamat email ini sudah terdaftar")
)

type CustomerType string

const (
	CustomerTypeUmum      CustomerType = "umum"
	CustomerTypeMahasiswa CustomerType = "mahasiswa"
)

func (t CustomerType) Label() string {
	switch t {
	case CustomerTypeMahasiswa:
		return "Mahasiswa (Sari Mulia)"
	case CustomerTypeUmum:
		return "Pelanggan Umum"
	default:
		return string(t)
	}
}

type Customer struct {
	ID                string       `json:"id"`
	Type              CustomerType `json:"type"`
	Name              string       `json:"name"`
	Phone             string       `json:"phone"`
	Email             string       `json:"email"`
	NIM               string       `json:"nim"`
	PasswordHash      string       `json:"-"`
	FirstDiscountUsed bool         `json:"first_discount_used"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

func (c *Customer) ShortName() string {
	parts := strings.Fields(c.Name)
	if len(parts) > 0 {
		return parts[0]
	}
	return c.Name
}

// CalculateFirstDiscount menghitung potongan 25% bagi mahasiswa pada pembelian pertama dengan total harga di atas Rp 20.000.
// Jika subtotal <= 20.000, diskon = 0.
// Jika subtotal > 20.000, diskon = 25% dari total harga pembelian.
func (c *Customer) CalculateFirstDiscount(subtotal int64) (int64, string) {
	if c == nil || c.Type != CustomerTypeMahasiswa || c.FirstDiscountUsed || subtotal <= 20000 {
		return 0, ""
	}

	discount := (subtotal * 25) / 100
	note := fmt.Sprintf("Diskon Mahasiswa Baru 25%% (NIM: %s)", c.NIM)
	return discount, note
}

type RegisterInput struct {
	Type     CustomerType
	Name     string
	Phone    string
	Email    string
	NIM      string
	Password string
}

type Session struct {
	Token      string
	CustomerID string
	ExpiresAt  time.Time
}

type Store struct {
	mu        sync.RWMutex
	customers map[string]*Customer // key: id
	sessions  map[string]*Session  // key: token
	seq       int
}

func NewStore() *Store {
	s := &Store{
		customers: make(map[string]*Customer),
		sessions:  make(map[string]*Session),
		seq:       0,
	}

	// Seed demo customers untuk kemudahan pengujian
	hashMahasiswa, _ := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.DefaultCost)
	custMhs := &Customer{
		ID:                "CUST-0001",
		Type:              CustomerTypeMahasiswa,
		Name:              "Budi Santoso",
		Phone:             "081234567890",
		Email:             "budi.mhs@example.com",
		NIM:               "211001001",
		PasswordHash:      string(hashMahasiswa),
		FirstDiscountUsed: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	s.customers[custMhs.ID] = custMhs

	hashUmum, _ := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.DefaultCost)
	custUmum := &Customer{
		ID:                "CUST-0002",
		Type:              CustomerTypeUmum,
		Name:              "Siti Rahma",
		Phone:             "089876543210",
		Email:             "siti.umum@example.com",
		NIM:               "",
		PasswordHash:      string(hashUmum),
		FirstDiscountUsed: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	s.customers[custUmum.ID] = custUmum
	s.seq = 2

	return s
}

var DefaultStore = NewStore()

func (s *Store) Register(input RegisterInput) (*Customer, error) {
	name := strings.TrimSpace(input.Name)
	if len(name) < 2 {
		return nil, ErrNameRequired
	}

	phone := cleanPhone(input.Phone)
	if len(phone) < 6 {
		return nil, ErrPhoneRequired
	}

	cType := input.Type
	if cType != CustomerTypeUmum && cType != CustomerTypeMahasiswa {
		return nil, ErrInvalidType
	}

	nim := strings.TrimSpace(input.NIM)
	if cType == CustomerTypeMahasiswa {
		if len(nim) < 3 {
			return nil, ErrNIMRequired
		}
	} else {
		nim = ""
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))

	if len(input.Password) < 6 {
		return nil, ErrPasswordTooShort
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.customers {
		if cleanPhone(existing.Phone) == phone {
			return nil, ErrPhoneAlreadyExists
		}
		if email != "" && strings.ToLower(existing.Email) == email {
			return nil, ErrEmailAlreadyExists
		}
		if cType == CustomerTypeMahasiswa && strings.EqualFold(existing.NIM, nim) {
			return nil, ErrNIMAlreadyExists
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("gagal memproses sandi: %w", err)
	}

	s.seq++
	id := fmt.Sprintf("CUST-%04d", s.seq)
	now := time.Now()

	cust := &Customer{
		ID:                id,
		Type:              cType,
		Name:              name,
		Phone:             phone,
		Email:             email,
		NIM:               nim,
		PasswordHash:      string(hash),
		FirstDiscountUsed: false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.customers[id] = cust
	return cust, nil
}

func (s *Store) Authenticate(identifier, password string) (*Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cleanID := strings.TrimSpace(identifier)
	phoneCandidate := cleanPhone(cleanID)

	var matched *Customer
	for _, cust := range s.customers {
		if (phoneCandidate != "" && cleanPhone(cust.Phone) == phoneCandidate) ||
			(cust.Type == CustomerTypeMahasiswa && strings.EqualFold(cust.NIM, cleanID)) {
			matched = cust
			break
		}
	}

	if matched == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(matched.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return matched, nil
}

func (s *Store) GetByID(id string) (*Customer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cust, ok := s.customers[id]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	return cust, nil
}

func (s *Store) MarkFirstDiscountUsed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cust, ok := s.customers[id]
	if !ok {
		return ErrCustomerNotFound
	}

	cust.FirstDiscountUsed = true
	cust.UpdatedAt = time.Now()
	return nil
}

func (s *Store) CreateSession(customerID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.customers[customerID]; !ok {
		return nil, ErrCustomerNotFound
	}

	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	sess := &Session{
		Token:      token,
		CustomerID: customerID,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}

	s.sessions[token] = sess
	return sess, nil
}

func (s *Store) GetCustomerBySession(token string) *Customer {
	if token == "" {
		return nil
	}

	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.customers[sess.CustomerID]
}

func (s *Store) DeleteSession(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func cleanPhone(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	s := sb.String()
	if strings.HasPrefix(s, "62") {
		s = "0" + strings.TrimPrefix(s, "62")
	}
	return s
}

func (s *Store) ExportCustomers() []*Customer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*Customer, 0, len(s.customers))
	for _, c := range s.customers {
		res = append(res, c)
	}
	return res
}

func (s *Store) ImportCustomers(list []*Customer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range list {
		if c == nil || c.ID == "" {
			continue
		}

		// Jangan menimpa hash sandi yang sudah valid (misalnya hasil pemuatan
		// dari MySQL) dengan hash kosong dari berkas lama.
		if existing, ok := s.customers[c.ID]; ok && c.PasswordHash == "" {
			c.PasswordHash = existing.PasswordHash
		}

		s.customers[c.ID] = c

		// Pulihkan urutan ID agar registrasi berikutnya tidak memakai ID
		// yang sudah ada dan menimpa akun pelanggan lain.
		if n, ok := parseCustomerSeq(c.ID); ok && n > s.seq {
			s.seq = n
		}
	}
}

func parseCustomerSeq(id string) (int, bool) {
	const prefix = "CUST-"
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
