package order

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"caffe-podjok-sendja/internal/catalog"
)

var (
	ErrEmptyCart             = errors.New("keranjang belanja masih kosong")
	ErrCustomerNameRequired  = errors.New("nama pelanggan wajib diisi minimal 2 karakter")
	ErrCustomerPhoneRequired = errors.New("nomor telepon wajib diisi")
	ErrTableNumberRequired   = errors.New("nomor meja wajib diisi untuk pesanan makan di tempat")
	ErrOrderNotFound         = errors.New("pesanan tidak ditemukan")
	ErrInvalidOrderType      = errors.New("tipe pesanan tidak valid")
	ErrInvalidPaymentMethod  = errors.New("metode pembayaran tidak valid")
)

type OrderType string

const (
	OrderTypeDineIn   OrderType = "dine_in"
	OrderTypeTakeaway OrderType = "takeaway"
)

type PaymentMethod string

const (
	PaymentMethodCash PaymentMethod = "cash"
	PaymentMethodQRIS PaymentMethod = "qris"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusPreparing      OrderStatus = "preparing"
	OrderStatusReady          OrderStatus = "ready"
	OrderStatusCompleted      OrderStatus = "completed"
	OrderStatusCancelled      OrderStatus = "cancelled"
)

type PaymentStatus string

const (
	PaymentStatusUnpaid PaymentStatus = "unpaid"
	PaymentStatusPaid   PaymentStatus = "paid"
)

type CartItem struct {
	ProductSlug string `json:"product_slug"`
	ProductName string `json:"product_name"`
	Variant     string `json:"variant"`
	UnitPrice   int64  `json:"unit_price"`
	Quantity    int    `json:"quantity"`
	Subtotal    int64  `json:"subtotal"`
}

type Cart struct {
	Items         []CartItem `json:"items"`
	TotalQuantity int        `json:"total_quantity"`
	Subtotal      int64      `json:"subtotal"`
}

func (c Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

type OrderItem struct {
	ProductSlug string `json:"product_slug"`
	ProductName string `json:"product_name"`
	Variant     string `json:"variant"`
	UnitPrice   int64  `json:"unit_price"`
	Quantity    int    `json:"quantity"`
	Subtotal    int64  `json:"subtotal"`
}

type Order struct {
	Code          string        `json:"code"`
	CustomerID    string        `json:"customer_id"`
	CustomerType  string        `json:"customer_type"`
	CustomerNIM   string        `json:"customer_nim"`
	CustomerName  string        `json:"customer_name"`
	CustomerPhone string        `json:"customer_phone"`
	OrderType     OrderType     `json:"order_type"`
	TableNumber   string        `json:"table_number"`
	Notes         string        `json:"notes"`
	Items         []OrderItem   `json:"items"`
	Subtotal      int64         `json:"subtotal"`
	Discount      int64         `json:"discount"`
	DiscountNote  string        `json:"discount_note"`
	Total         int64         `json:"total"`
	PaymentMethod PaymentMethod `json:"payment_method"`
	PaymentStatus PaymentStatus `json:"payment_status"`
	OrderStatus   OrderStatus   `json:"order_status"`
	QRISURL       string        `json:"qris_url,omitempty"`
	QRISString    string        `json:"qris_string,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type CreateOrderInput struct {
	CustomerID    string
	CustomerType  string
	CustomerNIM   string
	CustomerName  string
	CustomerPhone string
	OrderType     OrderType
	TableNumber   string
	Notes         string
	PaymentMethod PaymentMethod
	Discount      int64
	DiscountNote  string
	Cart          Cart
}

func (o *Order) StatusLabel() string {
	switch o.OrderStatus {
	case OrderStatusPendingPayment:
		return "Menunggu Pembayaran"
	case OrderStatusPaid:
		return "Pembayaran Diterima"
	case OrderStatusPreparing:
		return "Sedang Disiapkan"
	case OrderStatusReady:
		return "Siap Diambil"
	case OrderStatusCompleted:
		return "Selesai"
	case OrderStatusCancelled:
		return "Dibatalkan"
	default:
		return string(o.OrderStatus)
	}
}

func (o *Order) OrderTypeLabel() string {
	switch o.OrderType {
	case OrderTypeDineIn:
		if o.TableNumber != "" {
			return fmt.Sprintf("Makan di Tempat (Meja %s)", o.TableNumber)
		}
		return "Makan di Tempat"
	case OrderTypeTakeaway:
		return "Bawa Pulang (Takeaway)"
	default:
		return string(o.OrderType)
	}
}

func (o *Order) PaymentMethodLabel() string {
	switch o.PaymentMethod {
	case PaymentMethodCash:
		return "Tunai di Kasir"
	case PaymentMethodQRIS:
		return "QRIS"
	default:
		return string(o.PaymentMethod)
	}
}

func (o *Order) PaymentStatusLabel() string {
	switch o.PaymentStatus {
	case PaymentStatusPaid:
		return "Lunas"
	case PaymentStatusUnpaid:
		return "Belum Bayar"
	default:
		return string(o.PaymentStatus)
	}
}

type Store struct {
	mu     sync.RWMutex
	orders map[string]*Order
	seq    int
}

func NewStore() *Store {
	return &Store{
		orders: make(map[string]*Order),
		seq:    0,
	}
}

var DefaultStore = NewStore()

func (s *Store) CreateOrder(input CreateOrderInput) (*Order, error) {
	name := strings.TrimSpace(input.CustomerName)
	if len(name) < 2 {
		return nil, ErrCustomerNameRequired
	}

	phone := strings.TrimSpace(input.CustomerPhone)
	if len(phone) < 6 {
		return nil, ErrCustomerPhoneRequired
	}

	if input.OrderType != OrderTypeDineIn && input.OrderType != OrderTypeTakeaway {
		return nil, ErrInvalidOrderType
	}

	table := strings.TrimSpace(input.TableNumber)
	if input.OrderType == OrderTypeDineIn && table == "" {
		return nil, ErrTableNumberRequired
	}

	if input.PaymentMethod != PaymentMethodCash && input.PaymentMethod != PaymentMethodQRIS {
		return nil, ErrInvalidPaymentMethod
	}

	if input.Cart.IsEmpty() {
		return nil, ErrEmptyCart
	}

	// Periksa ketersediaan stok untuk setiap menu sebelum pesanan diproses
	for _, item := range input.Cart.Items {
		qty, tracked := catalog.DefaultStore.GetStock(item.ProductSlug)
		if tracked && qty < item.Quantity {
			return nil, fmt.Errorf("stok %s tidak mencukupi (tersisa %d porsi)", item.ProductName, qty)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now()
	code := fmt.Sprintf("PDS-%s-%04d", now.Format("060102"), s.seq)

	var orderItems []OrderItem
	var subtotal int64
	for _, item := range input.Cart.Items {
		itemSubtotal := item.UnitPrice * int64(item.Quantity)
		orderItems = append(orderItems, OrderItem{
			ProductSlug: item.ProductSlug,
			ProductName: item.ProductName,
			Variant:     item.Variant,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			Subtotal:    itemSubtotal,
		})
		subtotal += itemSubtotal
	}

	total := subtotal - input.Discount
	if total < 0 {
		total = 0
	}

	order := &Order{
		Code:          code,
		CustomerID:    input.CustomerID,
		CustomerType:  input.CustomerType,
		CustomerNIM:   input.CustomerNIM,
		CustomerName:  name,
		CustomerPhone: phone,
		OrderType:     input.OrderType,
		TableNumber:   table,
		Notes:         strings.TrimSpace(input.Notes),
		Items:         orderItems,
		Subtotal:      subtotal,
		Discount:      input.Discount,
		DiscountNote:  strings.TrimSpace(input.DiscountNote),
		Total:         total,
		PaymentMethod: input.PaymentMethod,
		PaymentStatus: PaymentStatusUnpaid,
		OrderStatus:   OrderStatusPendingPayment,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.orders[code] = order

	// Kurangi stok dari katalog
	for _, item := range input.Cart.Items {
		_ = catalog.DefaultStore.AdjustStock(item.ProductSlug, -item.Quantity, fmt.Sprintf("Pesanan %s", code))
	}

	return order, nil
}

func (s *Store) GetOrderByCode(code string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	order, ok := s.orders[cleanCode]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (s *Store) UpdateOrderStatus(code string, status OrderStatus, paymentStatus PaymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	order, ok := s.orders[cleanCode]
	if !ok {
		return ErrOrderNotFound
	}

	// Kembalikan stok jika pesanan dibatalkan dari status sebelumnya
	if status == OrderStatusCancelled && order.OrderStatus != OrderStatusCancelled {
		for _, item := range order.Items {
			_ = catalog.DefaultStore.AdjustStock(item.ProductSlug, item.Quantity, fmt.Sprintf("Pengembalian stok pesanan %s dibatalkan", cleanCode))
		}
	}

	order.OrderStatus = status
	order.PaymentStatus = paymentStatus
	order.UpdatedAt = time.Now()
	return nil
}

func (s *Store) CancelOrder(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	order, ok := s.orders[cleanCode]
	if !ok {
		return ErrOrderNotFound
	}

	if order.OrderStatus != OrderStatusPendingPayment {
		return errors.New("hanya pesanan berstatus Menunggu Pembayaran yang dapat dibatalkan")
	}

	order.OrderStatus = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	for _, item := range order.Items {
		_ = catalog.DefaultStore.AdjustStock(item.ProductSlug, item.Quantity, fmt.Sprintf("Pengembalian stok pesanan %s dibatalkan pelanggan", cleanCode))
	}

	return nil
}

func (s *Store) SetOrderQRIS(code string, qrisURL string, qrisString string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	if order, ok := s.orders[cleanCode]; ok {
		order.QRISURL = qrisURL
		order.QRISString = qrisString
		order.UpdatedAt = time.Now()
	}
}

type StoreMetrics struct {
	TotalOrders    int
	PendingPayment int
	InProcess      int
	ReadyForPickup int
	Completed      int
	ConfirmedSales int64
}

func (s *Store) GetMetrics() StoreMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var m StoreMetrics
	m.TotalOrders = len(s.orders)
	for _, o := range s.orders {
		switch o.OrderStatus {
		case OrderStatusPendingPayment:
			m.PendingPayment++
		case OrderStatusPaid, OrderStatusPreparing:
			m.InProcess++
		case OrderStatusReady:
			m.ReadyForPickup++
		case OrderStatusCompleted:
			m.Completed++
		}

		if o.PaymentStatus == PaymentStatusPaid {
			m.ConfirmedSales += o.Total
		}
	}
	return m
}

func (s *Store) ListOrdersDesc(filterStatus OrderStatus) []*Order {
	return s.ListOrdersFiltered("all", filterStatus)
}

func (s *Store) ListOrdersFiltered(period string, filterStatus OrderStatus) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	monthStr := now.Format("2006-01")

	result := make([]*Order, 0, len(s.orders))
	for _, o := range s.orders {
		if filterStatus != "" && o.OrderStatus != filterStatus {
			continue
		}
		switch period {
		case "today":
			if o.CreatedAt.Format("2006-01-02") != todayStr {
				continue
			}
		case "month":
			if o.CreatedAt.Format("2006-01") != monthStr {
				continue
			}
		}
		result = append(result, o)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

func (s *Store) ListOrders() []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Order, 0, len(s.orders))
	for _, order := range s.orders {
		result = append(result, order)
	}
	return result
}

func (s *Store) ListOrdersByCustomerID(customerID string) []*Order {
	if customerID == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Order
	for _, order := range s.orders {
		if order.CustomerID == customerID {
			result = append(result, order)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// BuildCart validates raw items from cookies against the authoritative catalog seed
func BuildCart(rawItems []struct {
	ProductSlug string `json:"product_slug"`
	Variant     string `json:"variant"`
	Quantity    int    `json:"quantity"`
}) Cart {
	categories := catalog.Seed()
	productMap := make(map[string]catalog.Product)
	for _, cat := range categories {
		for _, prod := range cat.Products {
			productMap[prod.Slug] = prod
		}
	}

	var items []CartItem
	var totalQty int
	var subtotal int64

	for _, raw := range rawItems {
		if raw.Quantity <= 0 {
			continue
		}
		prod, exists := productMap[raw.ProductSlug]
		if !exists {
			continue
		}

		price, err := prod.PriceFor(raw.Variant)
		if err != nil {
			continue
		}

		itemSubtotal := price * int64(raw.Quantity)
		items = append(items, CartItem{
			ProductSlug: prod.Slug,
			ProductName: prod.Name,
			Variant:     raw.Variant,
			UnitPrice:   price,
			Quantity:    raw.Quantity,
			Subtotal:    itemSubtotal,
		})
		totalQty += raw.Quantity
		subtotal += itemSubtotal
	}

	return Cart{
		Items:         items,
		TotalQuantity: totalQty,
		Subtotal:      subtotal,
	}
}

func (s *Store) ExportState() (map[string]*Order, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ordersCopy := make(map[string]*Order, len(s.orders))
	for k, v := range s.orders {
		ordersCopy[k] = v
	}
	return ordersCopy, s.seq
}

func (s *Store) ImportState(orders map[string]*Order, seq int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if orders != nil {
		s.orders = orders
	}
	if seq > s.seq {
		s.seq = seq
	}
}
