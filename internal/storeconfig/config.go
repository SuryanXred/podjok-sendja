package storeconfig

import (
	"strings"
	"sync"
)

type Config struct {
	IsOpen       bool   `json:"is_open"`
	OpeningHours string `json:"opening_hours"`
	ContactPhone string `json:"contact_phone"`
	TableCount   int    `json:"table_count"`
	Announcement string `json:"announcement"`
	Address      string `json:"address"`
}

type Store struct {
	mu  sync.RWMutex
	cfg Config
}

var DefaultStore = NewStore()

func NewStore() *Store {
	return &Store{
		cfg: Config{
			IsOpen:       true,
			OpeningHours: "09.00 - 22.00 WITA",
			ContactPhone: "0812-3456-7890",
			TableCount:   15,
			Announcement: "Selamat datang di Caffe Podjok Sendja. Potongan 25% untuk pemesanan pertama mahasiswa baru!",
			Address:      "Jl. Pramuka No. 2, Kelurahan Pemurus Luar, Kecamatan Banjarmasin Timur, Kota Banjarmasin, Kalimantan Selatan 70238",
		},
	}
}

func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Store) Set(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

func (s *Store) Update(isOpen bool, hours string, phone string, tables int, announcement string, address ...string) Config {
	s.mu.Lock()
	defer s.mu.Unlock()

	hours = strings.TrimSpace(hours)
	if hours != "" {
		s.cfg.OpeningHours = hours
	}

	phone = strings.TrimSpace(phone)
	if phone != "" {
		s.cfg.ContactPhone = phone
	}

	if tables > 0 {
		s.cfg.TableCount = tables
	}

	s.cfg.IsOpen = isOpen
	s.cfg.Announcement = strings.TrimSpace(announcement)

	if len(address) > 0 {
		addr := strings.TrimSpace(address[0])
		if addr != "" {
			s.cfg.Address = addr
		}
	}

	return s.cfg
}

func Get() Config {
	return DefaultStore.Get()
}

func Update(isOpen bool, hours string, phone string, tables int, announcement string, address ...string) Config {
	return DefaultStore.Update(isOpen, hours, phone, tables, announcement, address...)
}
