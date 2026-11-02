package database

import (
	"testing"
	"time"

	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/storeconfig"
)

func TestDatabaseConfigDSN(t *testing.T) {
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     "3306",
		User:     "root",
		Password: "",
		DBName:   "caffe_podjok_sendja",
	}

	dsn := cfg.DSN()
	expected := "root:@tcp(127.0.0.1:3306)/caffe_podjok_sendja?parseTime=true&charset=utf8mb4&loc=Local"
	if dsn != expected {
		t.Errorf("DSN = %q, want %q", dsn, expected)
	}
}

func TestDatabaseInitAndOperations(t *testing.T) {
	db, err := Init()
	if err != nil {
		t.Skipf("MySQL not available on local test environment (%v), skipping live MySQL test", err)
		return
	}
	defer db.Close()

	if !IsConnected() {
		t.Fatal("IsConnected() should return true after successful Init()")
	}

	// 1. Test Save & Update Store Config
	cfg := storeconfig.Config{
		IsOpen:       true,
		OpeningHours: "08.00 - 23.00 WITA",
		ContactPhone: "0812-9999-0000",
		TableCount:   20,
		Announcement: "Uji coba integrasi database MySQL.",
	}
	if err := UpdateStoreConfig(cfg); err != nil {
		t.Errorf("UpdateStoreConfig error: %v", err)
	}

	// 2. Test Save Customer
	cust := &customer.Customer{
		ID:                "CUST-TEST-001",
		Type:              customer.CustomerTypeMahasiswa,
		Name:              "Mahasiswa Uji Coba",
		Phone:             "081233445566",
		Email:             "mhs.test@sendja.com",
		NIM:               "211099999",
		PasswordHash:      "hashed_dummy_password",
		FirstDiscountUsed: false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := SaveCustomer(cust); err != nil {
		t.Errorf("SaveCustomer error: %v", err)
	}

	// 3. Test Save Order
	ord := &order.Order{
		Code:          "PDS-TEST-0001",
		CustomerID:    cust.ID,
		CustomerType:  string(cust.Type),
		CustomerNIM:   cust.NIM,
		CustomerName:  cust.Name,
		CustomerPhone: cust.Phone,
		OrderType:     order.OrderTypeDineIn,
		TableNumber:   "08",
		Notes:         "Uji integrasi",
		Items: []order.OrderItem{
			{
				ProductSlug: "americano",
				ProductName: "Americano",
				Variant:     "Ice",
				UnitPrice:   15000,
				Quantity:    2,
				Subtotal:    30000,
			},
		},
		Subtotal:      30000,
		Discount:      7500,
		DiscountNote:  "Diskon 25%",
		Total:         22500,
		PaymentMethod: order.PaymentMethodQRIS,
		PaymentStatus: order.PaymentStatusUnpaid,
		OrderStatus:   order.OrderStatusPendingPayment,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := SaveOrder(ord); err != nil {
		t.Errorf("SaveOrder error: %v", err)
	}

	// 4. Test Update Order Status
	if err := UpdateOrderStatus(ord.Code, string(order.OrderStatusPaid), string(order.PaymentStatusPaid)); err != nil {
		t.Errorf("UpdateOrderStatus error: %v", err)
	}

	// 5. Test Adjust Stock
	if err := AdjustStock("americano", 33, -2, "Pesanan PDS-TEST-0001"); err != nil {
		t.Errorf("AdjustStock error: %v", err)
	}

	// 6. Test Update Product Price
	if err := UpdateProductPrice("americano", 13000); err != nil {
		t.Errorf("UpdateProductPrice error: %v", err)
	}
}

