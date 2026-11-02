package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/storeconfig"
)

func TestPersistenceSaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pds_data_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	Init(tempDir)

	// Ubah konfigurasi toko
	storeconfig.Update(false, "10.00 - 20.00 WITA", "0811-9999-8888", 20, "Pengumuman uji coba persistensi.")
	SaveConfig()

	// Pastikan file config.json dibuat
	configPath := filepath.Join(tempDir, "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config.json not created: %v", err)
	}

	// Reset storeconfig ke default
	storeconfig.DefaultStore.Set(storeconfig.Config{
		IsOpen:       true,
		OpeningHours: "default",
	})

	// Re-init dengan direktori yang sama untuk memuat ulang
	Init(tempDir)

	loaded := storeconfig.Get()
	if loaded.IsOpen != false || loaded.OpeningHours != "10.00 - 20.00 WITA" || loaded.TableCount != 20 {
		t.Errorf("loaded config mismatch: %+v", loaded)
	}

	// Test order persistence
	SaveOrders()
	ordersPath := filepath.Join(tempDir, "orders.json")
	if _, err := os.Stat(ordersPath); err != nil {
		t.Fatalf("orders.json not created: %v", err)
	}

	_ = order.DefaultStore
}

