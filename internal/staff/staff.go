package staff

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrStaffNotFound      = errors.New("akun pegawai tidak ditemukan")
	ErrInvalidCredentials = errors.New("identitas atau kata sandi pegawai tidak cocok")
	ErrAccessForbidden    = errors.New("akses ditolak: hanya administrator yang memiliki hak akses halaman ini")
)

type Role string

const (
	RoleCashier Role = "cashier"
	RoleAdmin   Role = "admin"
)

func (r Role) Label() string {
	switch r {
	case RoleAdmin:
		return "Administrator"
	case RoleCashier:
		return "Kasir Operasional"
	default:
		return string(r)
	}
}

type StaffUser struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u *StaffUser) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

func (u *StaffUser) CanAccessCashier() bool {
	return u != nil && (u.Role == RoleCashier || u.Role == RoleAdmin)
}

func (u *StaffUser) CanAccessAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

type Session struct {
	Token     string
	StaffID   string
	ExpiresAt time.Time
}

type Store struct {
	mu       sync.RWMutex
	users    map[string]*StaffUser // key: staff ID
	sessions map[string]*Session   // key: token
}

var DefaultStore = NewStore()

func NewStore() *Store {
	s := &Store{
		users:    make(map[string]*StaffUser),
		sessions: make(map[string]*Session),
	}

	// 1. Akun Admin (Hak Akses Penuh: Admin Dashboard + Panel Kasir)
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	adminUser := &StaffUser{
		ID:           "STAFF-001",
		Username:     "admin",
		Email:        "admin@sendja.com",
		Name:         "Manajer Kafe",
		Role:         RoleAdmin,
		PasswordHash: string(adminHash),
		CreatedAt:    time.Now(),
	}
	s.users[adminUser.ID] = adminUser

	// 2. Akun Kasir (Hak Akses Terbatas: Hanya Panel Kasir)
	cashierHash, _ := bcrypt.GenerateFromPassword([]byte("kasir123"), bcrypt.DefaultCost)
	cashierUser := &StaffUser{
		ID:           "STAFF-002",
		Username:     "kasir",
		Email:        "kasir@sendja.com",
		Name:         "Kasir Meja Depan",
		Role:         RoleCashier,
		PasswordHash: string(cashierHash),
		CreatedAt:    time.Now(),
	}
	s.users[cashierUser.ID] = cashierUser

	return s
}

func (s *Store) Authenticate(identifier string, password string) (*StaffUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idLower := strings.ToLower(strings.TrimSpace(identifier))
	var matched *StaffUser

	for _, u := range s.users {
		if strings.ToLower(u.Username) == idLower {
			matched = u
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

func (s *Store) CreateSession(staffID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[staffID]
	if !exists {
		return nil, ErrStaffNotFound
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(raw)

	sess := &Session{
		Token:     token,
		StaffID:   user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Berlaku 24 jam untuk shift pegawai
	}
	s.sessions[token] = sess

	return sess, nil
}

func (s *Store) GetStaffBySession(token string) *StaffUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	sess, exists := s.sessions[token]
	if !exists {
		return nil
	}

	if time.Now().After(sess.ExpiresAt) {
		return nil
	}

	return s.users[sess.StaffID]
}

func (s *Store) DeleteSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *Store) AddOrUpdateUser(u *StaffUser) {
	if u == nil || u.ID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[u.ID] = u
}
