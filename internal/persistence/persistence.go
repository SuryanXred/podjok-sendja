package persistence

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"caffe-podjok-sendja/internal/catalog"
	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/storeconfig"
)

var (
	mu      sync.Mutex
	dataDir string
)

type CatalogState struct {
	Categories []catalog.Category      `json:"categories"`
	Stock      map[string]int          `json:"stock"`
	Movements  []catalog.StockMovement `json:"movements"`
}

type OrdersState struct {
	Orders map[string]*order.Order `json:"orders"`
	Seq    int                     `json:"seq"`
}

// Init menginisialisasi penyimpanan berbasis file pada direktori yang ditentukan.
func Init(dir string) {
	mu.Lock()
	defer mu.Unlock()

	if dir == "" {
		return
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[persistence] peringatan: gagal membuat direktori data %s: %v", dir, err)
		return
	}

	dataDir = dir

	// 1. Load config jika berkas ada
	loadConfig()

	// 2. Load catalog jika berkas ada
	loadCatalog()

	// 3. Load customers jika berkas ada
	loadCustomers()

	// 4. Load orders jika berkas ada
	loadOrders()

	log.Printf("[persistence] direktori penyimpanan aktif di: %s", dir)
}

func loadConfig() {
	path := filepath.Join(dataDir, "config.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var cfg storeconfig.Config
	if err := json.Unmarshal(bytes, &cfg); err == nil {
		storeconfig.DefaultStore.Set(cfg)
	}
}

func loadCatalog() {
	path := filepath.Join(dataDir, "catalog.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var state CatalogState
	if err := json.Unmarshal(bytes, &state); err == nil {
		catalog.DefaultStore.ImportState(state.Categories, state.Stock, state.Movements)
	}
}

func loadCustomers() {
	path := filepath.Join(dataDir, "customers.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var list []*customer.Customer
	if err := json.Unmarshal(bytes, &list); err == nil {
		customer.DefaultStore.ImportCustomers(list)
	}
}

func loadOrders() {
	path := filepath.Join(dataDir, "orders.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var state OrdersState
	if err := json.Unmarshal(bytes, &state); err == nil {
		order.DefaultStore.ImportState(state.Orders, state.Seq)
	}
}

// SaveConfig menyimpan konfigurasi toko ke disk.
func SaveConfig() {
	mu.Lock()
	defer mu.Unlock()

	if dataDir == "" {
		return
	}

	cfg := storeconfig.Get()
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dataDir, "config.json"), bytes, 0644)
}

// SaveCatalog menyimpan katalog, stok, dan pergerakan stok ke disk.
func SaveCatalog() {
	mu.Lock()
	defer mu.Unlock()

	if dataDir == "" {
		return
	}

	cats, stock, moves := catalog.DefaultStore.ExportState()
	state := CatalogState{
		Categories: cats,
		Stock:      stock,
		Movements:  moves,
	}

	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dataDir, "catalog.json"), bytes, 0644)
}

// SaveCustomers menyimpan seluruh akun pelanggan ke disk.
func SaveCustomers() {
	mu.Lock()
	defer mu.Unlock()

	if dataDir == "" {
		return
	}

	list := customer.DefaultStore.ExportCustomers()
	bytes, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dataDir, "customers.json"), bytes, 0644)
}

// SaveOrders menyimpan seluruh riwayat pesanan ke disk.
func SaveOrders() {
	mu.Lock()
	defer mu.Unlock()

	if dataDir == "" {
		return
	}

	orders, seq := order.DefaultStore.ExportState()
	state := OrdersState{
		Orders: orders,
		Seq:    seq,
	}

	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dataDir, "orders.json"), bytes, 0644)
}

// SaveAll menyimpan seluruh kondisi toko secara serentak.
func SaveAll() {
	SaveConfig()
	SaveCatalog()
	SaveCustomers()
	SaveOrders()
}

