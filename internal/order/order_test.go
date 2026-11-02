package order

import (
	"strings"
	"testing"

	"caffe-podjok-sendja/internal/catalog"
)

func TestBuildCartValidatesAgainstAuthoritativeCatalog(t *testing.T) {
	raw := []struct {
		ProductSlug string `json:"product_slug"`
		Variant     string `json:"variant"`
		Quantity    int    `json:"quantity"`
	}{
		{ProductSlug: "americano", Variant: "Ice", Quantity: 2},
		{ProductSlug: "french-fries", Variant: "", Quantity: 1},
		// Invalid items that must be filtered out
		{ProductSlug: "non-existent", Variant: "", Quantity: 1},
		{ProductSlug: "americano", Variant: "InvalidVariant", Quantity: 1},
		{ProductSlug: "espresso", Variant: "", Quantity: 0},
	}

	cart := BuildCart(raw)

	if cart.TotalQuantity != 3 {
		t.Errorf("TotalQuantity = %d, want 3", cart.TotalQuantity)
	}

	// Americano Ice = 15.000 * 2 = 30.000
	// French Fries = 12.000 * 1 = 12.000
	// Total = 42.000
	if cart.Subtotal != 42000 {
		t.Errorf("Subtotal = %d, want 42000", cart.Subtotal)
	}

	if len(cart.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(cart.Items))
	}

	if cart.Items[0].ProductName != "Americano" || cart.Items[0].UnitPrice != 15000 {
		t.Errorf("item 0 mismatch: %+v", cart.Items[0])
	}
}

func TestStoreCreateOrderValidation(t *testing.T) {
	store := NewStore()

	validCart := Cart{
		Items: []CartItem{
			{ProductSlug: "espresso", ProductName: "Espresso", UnitPrice: 10000, Quantity: 1, Subtotal: 10000},
		},
		TotalQuantity: 1,
		Subtotal:      10000,
	}

	// 1. Empty name
	_, err := store.CreateOrder(CreateOrderInput{
		CustomerName:  "",
		CustomerPhone: "08123456789",
		OrderType:     OrderTypeTakeaway,
		PaymentMethod: PaymentMethodCash,
		Cart:          validCart,
	})
	if err != ErrCustomerNameRequired {
		t.Errorf("expected ErrCustomerNameRequired, got %v", err)
	}

	// 2. Empty phone
	_, err = store.CreateOrder(CreateOrderInput{
		CustomerName:  "Budi",
		CustomerPhone: "123",
		OrderType:     OrderTypeTakeaway,
		PaymentMethod: PaymentMethodCash,
		Cart:          validCart,
	})
	if err != ErrCustomerPhoneRequired {
		t.Errorf("expected ErrCustomerPhoneRequired, got %v", err)
	}

	// 3. Dine-in without table number
	_, err = store.CreateOrder(CreateOrderInput{
		CustomerName:  "Budi",
		CustomerPhone: "08123456789",
		OrderType:     OrderTypeDineIn,
		TableNumber:   "",
		PaymentMethod: PaymentMethodCash,
		Cart:          validCart,
	})
	if err != ErrTableNumberRequired {
		t.Errorf("expected ErrTableNumberRequired, got %v", err)
	}

	// 4. Empty cart
	_, err = store.CreateOrder(CreateOrderInput{
		CustomerName:  "Budi",
		CustomerPhone: "08123456789",
		OrderType:     OrderTypeTakeaway,
		PaymentMethod: PaymentMethodCash,
		Cart:          Cart{},
	})
	if err != ErrEmptyCart {
		t.Errorf("expected ErrEmptyCart, got %v", err)
	}

	// 5. Successful creation
	order, err := store.CreateOrder(CreateOrderInput{
		CustomerName:  "Siti Rahma",
		CustomerPhone: "081234567890",
		OrderType:     OrderTypeDineIn,
		TableNumber:   "05",
		Notes:         "Sedikit gula",
		PaymentMethod: PaymentMethodQRIS,
		Cart:          validCart,
	})
	if err != nil {
		t.Fatalf("unexpected error creating order: %v", err)
	}

	if !strings.HasPrefix(order.Code, "PDS-") {
		t.Errorf("order code %q should start with PDS-", order.Code)
	}
	if order.Total != 10000 {
		t.Errorf("order total = %d, want 10000", order.Total)
	}
	if order.OrderStatus != OrderStatusPendingPayment {
		t.Errorf("order status = %v, want %v", order.OrderStatus, OrderStatusPendingPayment)
	}

	// 6. Retrieval by code
	retrieved, err := store.GetOrderByCode(order.Code)
	if err != nil {
		t.Fatalf("unexpected error retrieving order: %v", err)
	}
	if retrieved.CustomerName != "Siti Rahma" {
		t.Errorf("retrieved name = %q, want Siti Rahma", retrieved.CustomerName)
	}

	// Case insensitive lookup
	retrievedLower, err := store.GetOrderByCode(strings.ToLower(order.Code))
	if err != nil {
		t.Fatalf("unexpected error retrieving lowercase code: %v", err)
	}
	if retrievedLower.Code != order.Code {
		t.Errorf("retrieved code = %q, want %q", retrievedLower.Code, order.Code)
	}
}

func TestStoreStockLockAndRelease(t *testing.T) {
	store := NewStore()

	initialStock, _ := catalog.DefaultStore.GetStock("espresso")

	cart := Cart{
		Items: []CartItem{
			{ProductSlug: "espresso", ProductName: "Espresso", UnitPrice: 10000, Quantity: 2, Subtotal: 20000},
		},
		TotalQuantity: 2,
		Subtotal:      20000,
	}

	order, err := store.CreateOrder(CreateOrderInput{
		CustomerName:  "Andi",
		CustomerPhone: "081234567890",
		OrderType:     OrderTypeTakeaway,
		PaymentMethod: PaymentMethodCash,
		Cart:          cart,
	})
	if err != nil {
		t.Fatalf("unexpected error creating order: %v", err)
	}

	stockAfterOrder, _ := catalog.DefaultStore.GetStock("espresso")
	if stockAfterOrder != initialStock-2 {
		t.Errorf("stock after order = %d, want %d", stockAfterOrder, initialStock-2)
	}

	// Cancel the order -> stock must be released back
	err = store.CancelOrder(order.Code)
	if err != nil {
		t.Fatalf("unexpected error cancelling order: %v", err)
	}

	stockAfterCancel, _ := catalog.DefaultStore.GetStock("espresso")
	if stockAfterCancel != initialStock {
		t.Errorf("stock after cancel = %d, want %d", stockAfterCancel, initialStock)
	}
}
