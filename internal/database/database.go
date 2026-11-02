package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	"caffe-podjok-sendja/internal/catalog"
	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/order"
	"caffe-podjok-sendja/internal/staff"
	"caffe-podjok-sendja/internal/storeconfig"
)

var (
	mu       sync.RWMutex
	globalDB *sql.DB
	isOnline bool
)

// Config menyimpan kredensial koneksi basis data MySQL.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// LoadConfigFromEnv membaca pengaturan database dari berkas .env atau variabel lingkungan.
func LoadConfigFromEnv() Config {
	loadDotEnv(".env")

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = os.Getenv("MYSQLHOST")
	}
	if host == "" {
		host = "127.0.0.1"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = os.Getenv("MYSQLPORT")
	}
	if port == "" {
		port = "3306"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = os.Getenv("MYSQLUSER")
	}
	if user == "" {
		user = "root"
	}

	pass := os.Getenv("DB_PASSWORD")
	if pass == "" {
		pass = os.Getenv("MYSQLPASSWORD")
	}

	name := os.Getenv("DB_NAME")
	if name == "" {
		name = os.Getenv("MYSQLDATABASE")
	}
	if name == "" {
		name = "caffe_podjok_sendja"
	}

	return Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		DBName:   name,
	}
}

func loadDotEnv(filepath string) {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return
	}
	lines := strings.Split(string(bytes), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}

// DSN menghasilkan connection string format MySQL driver.
func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

// Init menginisialisasi koneksi MySQL, menjalankan migrasi skema tabel jika belum ada,
// dan menyinkronkan data antara database dan state aplikasi.
func Init() (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	cfg := LoadConfigFromEnv()
	dsn := cfg.DSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("[database] gagal membuka driver mysql: %v", err)
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Printf("[database] MySQL tidak terhubung (%s): %v. Berjalan dalam mode penyimpanan lokal.", dsn, err)
		return nil, err
	}

	globalDB = db
	isOnline = true
	log.Printf("[database] terhubung ke MySQL: %s:%s/%s", cfg.Host, cfg.Port, cfg.DBName)

	if err := createTables(db); err != nil {
		log.Printf("[database] gagal membuat tabel: %v", err)
		return db, err
	}

	if err := seedIfEmpty(db); err != nil {
		log.Printf("[database] peringatan saat seed data: %v", err)
	}

	if err := loadAllFromDB(db); err != nil {
		log.Printf("[database] gagal memuat data awal dari MySQL: %v", err)
	} else {
		log.Printf("[database] data katalog, pelanggan, pesanan, dan konfigurasi toko berhasil disinkronkan dari MySQL.")
	}

	return db, nil
}

// IsConnected memeriksa apakah koneksi database aktif.
func IsConnected() bool {
	mu.RLock()
	defer mu.RUnlock()
	return isOnline && globalDB != nil
}

// GetDB mengembalikan instance sql.DB aktif.
func GetDB() *sql.DB {
	mu.RLock()
	defer mu.RUnlock()
	return globalDB
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(50) PRIMARY KEY,
			username VARCHAR(50) NULL UNIQUE,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(150) NULL,
			phone VARCHAR(20) NULL,
			nim VARCHAR(30) NULL,
			password_hash VARCHAR(255) NOT NULL,
			role ENUM('customer', 'cashier', 'admin') NOT NULL DEFAULT 'customer',
			customer_type ENUM('umum', 'mahasiswa') NOT NULL DEFAULT 'umum',
			first_discount_used BOOLEAN NOT NULL DEFAULT FALSE,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS categories (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(80) NOT NULL,
			slug VARCHAR(100) NOT NULL UNIQUE,
			sort_order INT NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS products (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			category_slug VARCHAR(100) NOT NULL,
			name VARCHAR(120) NOT NULL,
			slug VARCHAR(150) NOT NULL UNIQUE,
			base_price BIGINT NOT NULL,
			track_stock BOOLEAN NOT NULL DEFAULT TRUE,
			stock_qty INT NOT NULL DEFAULT 0,
			low_stock_threshold INT NOT NULL DEFAULT 5,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			CONSTRAINT fk_products_category FOREIGN KEY (category_slug) REFERENCES categories(slug) ON DELETE RESTRICT
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS product_variants (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			product_slug VARCHAR(150) NOT NULL,
			name VARCHAR(60) NOT NULL,
			price_delta BIGINT NOT NULL DEFAULT 0,
			sort_order INT NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			CONSTRAINT fk_variants_product FOREIGN KEY (product_slug) REFERENCES products(slug) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS orders (
			code VARCHAR(30) PRIMARY KEY,
			customer_id VARCHAR(50) NULL,
			customer_type VARCHAR(30) NOT NULL DEFAULT 'umum',
			customer_nim VARCHAR(30) NULL,
			customer_name VARCHAR(100) NOT NULL,
			customer_phone VARCHAR(20) NOT NULL,
			order_type VARCHAR(20) NOT NULL,
			table_number VARCHAR(10) NULL,
			notes TEXT NULL,
			subtotal BIGINT NOT NULL,
			discount BIGINT NOT NULL DEFAULT 0,
			discount_note VARCHAR(255) NULL,
			total BIGINT NOT NULL,
			payment_method VARCHAR(20) NOT NULL,
			payment_status VARCHAR(20) NOT NULL,
			order_status VARCHAR(30) NOT NULL,
			qris_url TEXT NULL,
			qris_string TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_orders_status (order_status),
			INDEX idx_orders_created (created_at),
			CONSTRAINT fk_orders_user FOREIGN KEY (customer_id) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS order_items (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			order_code VARCHAR(30) NOT NULL,
			product_slug VARCHAR(150) NOT NULL,
			product_name VARCHAR(120) NOT NULL,
			variant_name VARCHAR(60) NULL,
			unit_price BIGINT NOT NULL,
			quantity INT NOT NULL,
			subtotal BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_items_order FOREIGN KEY (order_code) REFERENCES orders(code) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS stock_movements (
			id INT AUTO_INCREMENT PRIMARY KEY,
			product_slug VARCHAR(150) NOT NULL,
			product_name VARCHAR(120) NOT NULL,
			change_qty INT NOT NULL,
			reason VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,

		`CREATE TABLE IF NOT EXISTS store_settings (
			id INT PRIMARY KEY DEFAULT 1,
			is_open BOOLEAN NOT NULL DEFAULT TRUE,
			opening_hours VARCHAR(100) NOT NULL,
			contact_phone VARCHAR(50) NOT NULL,
			table_count INT NOT NULL DEFAULT 15,
			announcement TEXT NULL,
			address VARCHAR(255) NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func seedIfEmpty(db *sql.DB) error {
	// 1. Seed Categories & Products
	var catCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&catCount)
	if catCount == 0 {
		initialCategories := catalog.Seed()
		now := time.Now()

		for _, cat := range initialCategories {
			_, err := db.Exec("INSERT INTO categories (name, slug, sort_order) VALUES (?, ?, ?)",
				cat.Name, cat.Slug, cat.SortOrder)
			if err != nil {
				return err
			}

			for _, prod := range cat.Products {
				initialStock := 25
				if cat.Slug == "coffee" {
					initialStock = 35
				} else if cat.Slug == "food" {
					initialStock = 20
				}

				_, err := db.Exec("INSERT INTO products (category_slug, name, slug, base_price, stock_qty) VALUES (?, ?, ?, ?, ?)",
					cat.Slug, prod.Name, prod.Slug, prod.BasePrice, initialStock)
				if err != nil {
					return err
				}

				for vIdx, v := range prod.Variants {
					_, err := db.Exec("INSERT INTO product_variants (product_slug, name, price_delta, sort_order) VALUES (?, ?, ?, ?)",
						prod.Slug, v.Name, v.PriceDelta, vIdx)
					if err != nil {
						return err
					}
				}

				_, _ = db.Exec("INSERT INTO stock_movements (product_slug, product_name, change_qty, reason, created_at) VALUES (?, ?, ?, ?, ?)",
					prod.Slug, prod.Name, initialStock, "Stok awal pembukaan kafe", now)
			}
		}
	}

	// 2. Seed Users (Staff & Demo Customers)
	var userCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		cashierHash, _ := bcrypt.GenerateFromPassword([]byte("kasir123"), bcrypt.DefaultCost)
		mhsHash, _ := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.DefaultCost)
		umumHash, _ := bcrypt.GenerateFromPassword([]byte("rahasia123"), bcrypt.DefaultCost)

		_, _ = db.Exec(`INSERT INTO users (id, username, name, email, role, password_hash) VALUES 
			('STAFF-001', 'admin', 'Manajer Kafe', 'admin@sendja.com', 'admin', ?),
			('STAFF-002', 'kasir', 'Kasir Meja Depan', 'kasir@sendja.com', 'cashier', ?)`,
			string(adminHash), string(cashierHash))

		_, _ = db.Exec(`INSERT INTO users (id, name, phone, email, nim, password_hash, role, customer_type, first_discount_used) VALUES 
			('CUST-0001', 'Budi Santoso', '081234567890', 'budi.mhs@example.com', '211001001', ?, 'customer', 'mahasiswa', FALSE),
			('CUST-0002', 'Siti Rahma', '089876543210', 'siti.umum@example.com', NULL, ?, 'customer', 'umum', FALSE)`,
			string(mhsHash), string(umumHash))
	}

	// 3. Seed Store Settings
	var settingsCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM store_settings WHERE id = 1").Scan(&settingsCount)
	if settingsCount == 0 {
		cfg := storeconfig.Get()
		_, _ = db.Exec(`INSERT INTO store_settings (id, is_open, opening_hours, contact_phone, table_count, announcement, address) 
			VALUES (1, ?, ?, ?, ?, ?, ?)`,
			cfg.IsOpen, cfg.OpeningHours, cfg.ContactPhone, cfg.TableCount, cfg.Announcement, cfg.Address)
	}

	return nil
}

func loadAllFromDB(db *sql.DB) error {
	// 1. Load Categories, Products, Variants, Stock
	catRows, err := db.Query("SELECT slug, name, sort_order FROM categories ORDER BY sort_order ASC")
	if err == nil {
		defer catRows.Close()
		var categories []catalog.Category
		for catRows.Next() {
			var c catalog.Category
			if err := catRows.Scan(&c.Slug, &c.Name, &c.SortOrder); err == nil {
				categories = append(categories, c)
			}
		}

		stockMap := make(map[string]int)
		for i := range categories {
			prodRows, err := db.Query("SELECT slug, name, base_price, stock_qty FROM products WHERE category_slug = ? ORDER BY id ASC", categories[i].Slug)
			if err == nil {
				var prods []catalog.Product
				for prodRows.Next() {
					var p catalog.Product
					var stockQty int
					if err := prodRows.Scan(&p.Slug, &p.Name, &p.BasePrice, &stockQty); err == nil {
						stockMap[p.Slug] = stockQty

						vRows, err := db.Query("SELECT name, price_delta FROM product_variants WHERE product_slug = ? ORDER BY sort_order ASC", p.Slug)
						if err == nil {
							var variants []catalog.Variant
							for vRows.Next() {
								var v catalog.Variant
								if err := vRows.Scan(&v.Name, &v.PriceDelta); err == nil {
									variants = append(variants, v)
								}
							}
							vRows.Close()
							p.Variants = variants
						}
						prods = append(prods, p)
					}
				}
				prodRows.Close()
				categories[i].Products = prods
			}
		}

		moveRows, err := db.Query("SELECT id, product_slug, product_name, change_qty, reason, created_at FROM stock_movements ORDER BY id ASC")
		var movements []catalog.StockMovement
		if err == nil {
			for moveRows.Next() {
				var m catalog.StockMovement
				if err := moveRows.Scan(&m.ID, &m.ProductSlug, &m.ProductName, &m.Change, &m.Reason, &m.CreatedAt); err == nil {
					movements = append(movements, m)
				}
			}
			moveRows.Close()
		}

		if len(categories) > 0 {
			catalog.DefaultStore.ImportState(categories, stockMap, movements)
		}
	}

	// 2. Load Customers
	custRows, err := db.Query("SELECT id, name, phone, COALESCE(email, ''), COALESCE(nim, ''), password_hash, customer_type, first_discount_used, created_at, updated_at FROM users WHERE role = 'customer'")
	if err == nil {
		defer custRows.Close()
		var customers []*customer.Customer
		for custRows.Next() {
			c := &customer.Customer{}
			var cType string
			if err := custRows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.NIM, &c.PasswordHash, &cType, &c.FirstDiscountUsed, &c.CreatedAt, &c.UpdatedAt); err == nil {
				c.Type = customer.CustomerType(cType)
				customers = append(customers, c)
			}
		}
		if len(customers) > 0 {
			customer.DefaultStore.ImportCustomers(customers)
		}
	}

	// 3. Load Staff Users
	staffRows, err := db.Query("SELECT id, username, COALESCE(email, ''), name, role, password_hash, created_at FROM users WHERE role IN ('cashier', 'admin')")
	if err == nil {
		defer staffRows.Close()
		for staffRows.Next() {
			var u staff.StaffUser
			var roleStr string
			if err := staffRows.Scan(&u.ID, &u.Username, &u.Email, &u.Name, &roleStr, &u.PasswordHash, &u.CreatedAt); err == nil {
				u.Role = staff.Role(roleStr)
				staff.DefaultStore.AddOrUpdateUser(&u)
			}
		}
	}

	// 4. Load Orders & Order Items
	orderRows, err := db.Query("SELECT code, COALESCE(customer_id, ''), customer_type, COALESCE(customer_nim, ''), customer_name, customer_phone, order_type, COALESCE(table_number, ''), COALESCE(notes, ''), subtotal, discount, COALESCE(discount_note, ''), total, payment_method, payment_status, order_status, COALESCE(qris_url, ''), COALESCE(qris_string, ''), created_at, updated_at FROM orders")
	if err == nil {
		defer orderRows.Close()
		ordersMap := make(map[string]*order.Order)
		maxSeq := 0
		for orderRows.Next() {
			o := &order.Order{}
			var oType, pMethod, pStatus, oStatus string
			if err := orderRows.Scan(&o.Code, &o.CustomerID, &o.CustomerType, &o.CustomerNIM, &o.CustomerName, &o.CustomerPhone, &oType, &o.TableNumber, &o.Notes, &o.Subtotal, &o.Discount, &o.DiscountNote, &o.Total, &pMethod, &pStatus, &oStatus, &o.QRISURL, &o.QRISString, &o.CreatedAt, &o.UpdatedAt); err == nil {
				o.OrderType = order.OrderType(oType)
				o.PaymentMethod = order.PaymentMethod(pMethod)
				o.PaymentStatus = order.PaymentStatus(pStatus)
				o.OrderStatus = order.OrderStatus(oStatus)

				// Ambil items
				itemRows, err := db.Query("SELECT product_slug, product_name, COALESCE(variant_name, ''), unit_price, quantity, subtotal FROM order_items WHERE order_code = ?", o.Code)
				if err == nil {
					for itemRows.Next() {
						var item order.OrderItem
						if err := itemRows.Scan(&item.ProductSlug, &item.ProductName, &item.Variant, &item.UnitPrice, &item.Quantity, &item.Subtotal); err == nil {
							o.Items = append(o.Items, item)
						}
					}
					itemRows.Close()
				}
				ordersMap[o.Code] = o
			}
		}
		if len(ordersMap) > 0 {
			order.DefaultStore.ImportState(ordersMap, maxSeq)
		}
	}

	// 5. Load Store Settings
	var cfg storeconfig.Config
	err = db.QueryRow("SELECT is_open, opening_hours, contact_phone, table_count, COALESCE(announcement, ''), COALESCE(address, '') FROM store_settings WHERE id = 1").
		Scan(&cfg.IsOpen, &cfg.OpeningHours, &cfg.ContactPhone, &cfg.TableCount, &cfg.Announcement, &cfg.Address)
	if err == nil {
		if cfg.Address == "" {
			cfg.Address = "Jl. Pramuka No. 2, Kelurahan Pemurus Luar, Kecamatan Banjarmasin Timur, Kota Banjarmasin, Kalimantan Selatan 70238"
		}
		storeconfig.DefaultStore.Set(cfg)
	}

	return nil
}

// SaveCustomer menyimpan atau memperbarui akun pelanggan di database.
func SaveCustomer(c *customer.Customer) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	query := `INSERT INTO users (id, name, phone, email, nim, password_hash, role, customer_type, first_discount_used, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'customer', ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			name = VALUES(name),
			phone = VALUES(phone),
			email = VALUES(email),
			nim = VALUES(nim),
			password_hash = VALUES(password_hash),
			first_discount_used = VALUES(first_discount_used),
			updated_at = VALUES(updated_at)`

	_, err := db.Exec(query,
		c.ID, c.Name, c.Phone, c.Email, c.NIM, c.PasswordHash, string(c.Type), c.FirstDiscountUsed, c.CreatedAt, c.UpdatedAt)
	return err
}

// SaveOrder menyimpan pesanan baru beserta rincian itemnya dalam satu transaksi MySQL.
func SaveOrder(o *order.Order) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryOrder := `INSERT INTO orders (code, customer_id, customer_type, customer_nim, customer_name, customer_phone, order_type, table_number, notes, subtotal, discount, discount_note, total, payment_method, payment_status, order_status, qris_url, qris_string, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			payment_status = VALUES(payment_status),
			order_status = VALUES(order_status),
			qris_url = VALUES(qris_url),
			qris_string = VALUES(qris_string),
			updated_at = VALUES(updated_at)`

	var custID interface{}
	if o.CustomerID != "" {
		custID = o.CustomerID
	}

	_, err = tx.Exec(queryOrder,
		o.Code, custID, o.CustomerType, o.CustomerNIM, o.CustomerName, o.CustomerPhone,
		string(o.OrderType), o.TableNumber, o.Notes, o.Subtotal, o.Discount, o.DiscountNote,
		o.Total, string(o.PaymentMethod), string(o.PaymentStatus), string(o.OrderStatus),
		o.QRISURL, o.QRISString, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert items jika belum ada
	for _, item := range o.Items {
		queryItem := `INSERT INTO order_items (order_code, product_slug, product_name, variant_name, unit_price, quantity, subtotal)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, _ = tx.Exec(queryItem, o.Code, item.ProductSlug, item.ProductName, item.Variant, item.UnitPrice, item.Quantity, item.Subtotal)
	}

	return tx.Commit()
}

// UpdateOrderStatus memperbarui status pesanan dan pembayaran di MySQL.
func UpdateOrderStatus(code string, status string, paymentStatus string) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	query := `UPDATE orders SET order_status = ?, payment_status = ?, updated_at = NOW() WHERE code = ?`
	_, err := db.Exec(query, status, paymentStatus, strings.ToUpper(strings.TrimSpace(code)))
	return err
}

// AdjustStock mencatat perubahan stok dan mutasi stok di MySQL.
func AdjustStock(slug string, newStock int, change int, reason string) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE products SET stock_qty = ?, updated_at = NOW() WHERE slug = ?", newStock, slug)
	if err != nil {
		return err
	}

	var prodName string
	_ = tx.QueryRow("SELECT name FROM products WHERE slug = ?", slug).Scan(&prodName)
	if prodName == "" {
		prodName = slug
	}

	_, err = tx.Exec("INSERT INTO stock_movements (product_slug, product_name, change_qty, reason, created_at) VALUES (?, ?, ?, ?, NOW())",
		slug, prodName, change, reason)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateProductPrice memperbarui harga dasar produk di MySQL.
func UpdateProductPrice(slug string, newPrice int64) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	_, err := db.Exec("UPDATE products SET base_price = ?, updated_at = NOW() WHERE slug = ?", newPrice, slug)
	return err
}

// AddProduct menambahkan menu baru beserta variannya ke MySQL.
func AddProduct(categorySlug string, p *catalog.Product, stockQty int) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO products (category_slug, name, slug, base_price, stock_qty) VALUES (?, ?, ?, ?, ?)",
		categorySlug, p.Name, p.Slug, p.BasePrice, stockQty)
	if err != nil {
		return err
	}

	for idx, v := range p.Variants {
		_, err := tx.Exec("INSERT INTO product_variants (product_slug, name, price_delta, sort_order) VALUES (?, ?, ?, ?)",
			p.Slug, v.Name, v.PriceDelta, idx)
		if err != nil {
			return err
		}
	}

	_, _ = tx.Exec("INSERT INTO stock_movements (product_slug, product_name, change_qty, reason, created_at) VALUES (?, ?, ?, ?, NOW())",
		p.Slug, p.Name, stockQty, "Penambahan menu baru oleh admin")

	return tx.Commit()
}

// UpdateStoreConfig memperbarui pengaturan toko di MySQL.
func UpdateStoreConfig(cfg storeconfig.Config) error {
	db := GetDB()
	if db == nil {
		return nil
	}

	query := `INSERT INTO store_settings (id, is_open, opening_hours, contact_phone, table_count, announcement, address) 
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			is_open = VALUES(is_open),
			opening_hours = VALUES(opening_hours),
			contact_phone = VALUES(contact_phone),
			table_count = VALUES(table_count),
			announcement = VALUES(announcement),
			address = VALUES(address),
			updated_at = NOW()`

	_, err := db.Exec(query, cfg.IsOpen, cfg.OpeningHours, cfg.ContactPhone, cfg.TableCount, cfg.Announcement, cfg.Address)
	return err
}
