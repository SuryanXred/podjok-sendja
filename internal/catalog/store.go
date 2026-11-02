package catalog

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrProductNotFound = errors.New("produk tidak ditemukan")
	ErrInvalidPrice    = errors.New("harga harus lebih besar dari 0")
	ErrNameRequired    = errors.New("nama menu wajib diisi minimal 2 karakter")
)

type StockItem struct {
	ProductSlug       string
	ProductName       string
	CategorySlug      string
	CategoryName      string
	Qty               int
	LowStockThreshold int
}

func (s StockItem) StatusLabel() string {
	if s.Qty <= 0 {
		return "Habis"
	}
	if s.Qty <= s.LowStockThreshold {
		return "Menipis"
	}
	return "Aman"
}

func (s StockItem) StatusCode() string {
	if s.Qty <= 0 {
		return "empty"
	}
	if s.Qty <= s.LowStockThreshold {
		return "low"
	}
	return "safe"
}

type StockMovement struct {
	ID          int
	ProductSlug string
	ProductName string
	Change      int
	Reason      string
	CreatedAt   time.Time
}

type Store struct {
	mu         sync.RWMutex
	categories []Category
	stock      map[string]int // key: product slug -> quantity
	movements  []StockMovement
	nextMoveID int
}

var DefaultStore = NewStore()

func NewStore() *Store {
	categories := Seed()
	stockMap := make(map[string]int)
	initialMovements := make([]StockMovement, 0)

	// Inisialisasi stok awal yang realistis untuk setiap menu (25 - 40 porsi)
	moveID := 1
	now := time.Now()
	for _, cat := range categories {
		for _, prod := range cat.Products {
			initialQty := 25
			if cat.Slug == "coffee" {
				initialQty = 35
			} else if cat.Slug == "food" {
				initialQty = 20
			}
			stockMap[prod.Slug] = initialQty
			initialMovements = append(initialMovements, StockMovement{
				ID:          moveID,
				ProductSlug: prod.Slug,
				ProductName: prod.Name,
				Change:      initialQty,
				Reason:      "Stok awal pembukaan kafe",
				CreatedAt:   now.Add(-2 * time.Hour),
			})
			moveID++
		}
	}

	return &Store{
		categories: categories,
		stock:      stockMap,
		movements:  initialMovements,
		nextMoveID: moveID,
	}
}

func (s *Store) GetCategories() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Category, len(s.categories))
	copy(result, s.categories)
	return result
}

func (s *Store) AddProduct(categorySlug string, name string, basePrice int64, variants []Variant) (*Product, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 {
		return nil, ErrNameRequired
	}
	if basePrice <= 0 {
		return nil, ErrInvalidPrice
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	catIndex := -1
	for i, c := range s.categories {
		if c.Slug == categorySlug {
			catIndex = i
			break
		}
	}
	if catIndex == -1 {
		return nil, errors.New("kategori tidak ditemukan")
	}

	// Generate slug unik
	baseSlug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	slug := baseSlug
	counter := 1
	slugExists := true
	for slugExists {
		slugExists = false
		for _, cat := range s.categories {
			for _, prod := range cat.Products {
				if prod.Slug == slug {
					slugExists = true
					slug = fmt.Sprintf("%s-%d", baseSlug, counter)
					counter++
					break
				}
			}
		}
	}

	newProduct := Product{
		Slug:      slug,
		Name:      name,
		BasePrice: basePrice,
		Variants:  variants,
	}

	s.categories[catIndex].Products = append(s.categories[catIndex].Products, newProduct)
	s.stock[slug] = 20 // Default stok awal untuk produk baru
	s.movements = append(s.movements, StockMovement{
		ID:          s.nextMoveID,
		ProductSlug: slug,
		ProductName: name,
		Change:      20,
		Reason:      "Penambahan menu baru oleh admin",
		CreatedAt:   time.Now(),
	})
	s.nextMoveID++

	return &newProduct, nil
}

func (s *Store) UpdateProductPrice(slug string, newPrice int64) error {
	if newPrice <= 0 {
		return ErrInvalidPrice
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for catIdx := range s.categories {
		for prodIdx := range s.categories[catIdx].Products {
			if s.categories[catIdx].Products[prodIdx].Slug == slug {
				s.categories[catIdx].Products[prodIdx].BasePrice = newPrice
				return nil
			}
		}
	}

	return ErrProductNotFound
}

func (s *Store) GetStock(slug string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	qty, ok := s.stock[slug]
	return qty, ok
}

func (s *Store) GetStockList() []StockItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []StockItem
	for _, cat := range s.categories {
		for _, prod := range cat.Products {
			qty := s.stock[prod.Slug]
			list = append(list, StockItem{
				ProductSlug:       prod.Slug,
				ProductName:       prod.Name,
				CategorySlug:      cat.Slug,
				CategoryName:      cat.Name,
				Qty:               qty,
				LowStockThreshold: 5,
			})
		}
	}
	return list
}

func (s *Store) AdjustStock(slug string, change int, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Penyesuaian stok admin"
	}

	var prodName string
	found := false
	for _, cat := range s.categories {
		for _, prod := range cat.Products {
			if prod.Slug == slug {
				prodName = prod.Name
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return ErrProductNotFound
	}

	current := s.stock[slug]
	newQty := current + change
	if newQty < 0 {
		newQty = 0
	}
	s.stock[slug] = newQty

	s.movements = append(s.movements, StockMovement{
		ID:          s.nextMoveID,
		ProductSlug: slug,
		ProductName: prodName,
		Change:      change,
		Reason:      reason,
		CreatedAt:   time.Now(),
	})
	s.nextMoveID++

	return nil
}

func (s *Store) GetStockMovements(limit int) []StockMovement {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.movements)
	if total == 0 {
		return nil
	}

	count := limit
	if count <= 0 || count > total {
		count = total
	}

	result := make([]StockMovement, 0, count)
	for i := total - 1; i >= total-count; i-- {
		result = append(result, s.movements[i])
	}
	return result
}

func (s *Store) ExportState() ([]Category, map[string]int, []StockMovement) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cats := make([]Category, len(s.categories))
	copy(cats, s.categories)

	stockCopy := make(map[string]int, len(s.stock))
	for k, v := range s.stock {
		stockCopy[k] = v
	}

	moves := make([]StockMovement, len(s.movements))
	copy(moves, s.movements)

	return cats, stockCopy, moves
}

func (s *Store) ImportState(cats []Category, stock map[string]int, moves []StockMovement) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(cats) > 0 {
		s.categories = cats
	}
	if len(stock) > 0 {
		s.stock = stock
	}
	if len(moves) > 0 {
		s.movements = moves
		maxID := 1
		for _, m := range moves {
			if m.ID >= maxID {
				maxID = m.ID + 1
			}
		}
		s.nextMoveID = maxID
	}
}
