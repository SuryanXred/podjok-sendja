# LAPORAN PENGUJIAN REKONSILIASI TRANSAKSI DAN AUDIT STOK
**Sistem Web Order & Manajemen Operasional Caffe Podjok Sendja (Kafe Sarmul)**  
**Universitas Sari Mulia Banjarmasin**

---

## 1. Parameter dan Metodologi Pengujian
Pengujian ini bertujuan untuk memvalidasi keakuratan perhitungan finansial (subtotal, diskon mahasiswa baru 25% dengan syarat total belanja di atas Rp 20.000, total bayar) serta integritas pengurangan dan pengembalian stok otomatis pada sistem saat pesanan selesai maupun dibatalkan.

* **Jumlah Pesanan Uji:** 15 pesanan
* **Variasi Pengujian:**
  * Kategori Produk: Coffee, Tea & Milk, Food, Snack (mencakup varian Hot dan Ice).
  * Kuantitas: 1, 2, hingga 3 porsi per menu.
  * Tipe Layanan: *Dine In* (dengan nomor meja) dan *Takeaway*.
  * Akun Pelanggan: Mahasiswa Baru (klaim diskon), Mahasiswa Belanja $\le$ Rp 20.000, Mahasiswa Pembelian Kedua, dan Pelanggan Umum.
  * Metode Pembayaran: Tunai di Kasir dan QRIS Dinamis (Simulasi).
  * Status Akhir: 12 pesanan selesai (*Completed/Paid*) dan 3 pesanan dibatalkan (*Cancelled*).

---

## 2. Pencatatan Stok Awal Produk (Data dari Dasbor Admin / `catalog.json`)

Berikut adalah pencatatan stok fisik awal 17 produk katalog Caffe Podjok Sendja sebelum pengujian dilakukan:

| No | Kode Produk (*Slug*) | Nama Produk | Kategori | Harga Dasar | Stok Awal |
| :-: | :--- | :--- | :--- | :---: | :---: |
| 1 | `americano` | Americano | Coffee | - | **33 porsi** |
| 2 | `espresso` | Espresso | Coffee | - | **34 porsi** |
| 3 | `caramel` | Caramel | Coffee | - | **34 porsi** |
| 4 | `vanilla-latte` | Vanilla Latte | Coffee | - | **35 porsi** |
| 5 | `brown-sugar` | Brown Sugar | Coffee | - | **35 porsi** |
| 6 | `spanish-latte` | Spanish Latte | Coffee | - | **35 porsi** |
| 7 | `ginger-tea` | Ginger Tea | Tea & Milk | - | **25 porsi** |
| 8 | `lemongrass` | Lemongrass | Tea & Milk | - | **25 porsi** |
| 9 | `choco-milk` | Choco Milk | Tea & Milk | - | **25 porsi** |
| 10 | `strawberry-milk` | Strawberry Milk | Tea & Milk | - | **25 porsi** |
| 11 | `matcha-latte` | Matcha Latte | Tea & Milk | - | **25 porsi** |
| 12 | `mie-aneka-rasa` | Mie Rebus/Goreng Aneka Rasa | Food | - | **20 porsi** |
| 13 | `mie-rebus-spesial` | Mie Rebus Spesial | Food | - | **20 porsi** |
| 14 | `mie-goreng-spesial` | Mie Goreng Spesial | Food | - | **20 porsi** |
| 15 | `telur` | Telur Rebus/Goreng | Food | - | **20 porsi** |
| 16 | `french-fries` | French Fries | Snack | - | **24 porsi** |
| 17 | `mix-platter` | Mix Platter (Sosis dan Kentang) | Snack | - | **25 porsi** |

---

## 3. Tabel Rekonsiliasi Perhitungan Manual vs Sistem (Ekspor Transaksi)

Tabel berikut menyandingkan perhitungan manual (*ground truth*) dengan nilai keluaran sistem aplikasi untuk memverifikasi ada/tidaknya selisih nominal:

| No | Kode Pesanan | Pelanggan & NIM | Layanan & Meja | Metode Bayar | Rincian Menu & Varian | Subtotal | Diskon (25%) | Total Manual | Total Sistem | Selisih | Status Pesanan | Kesesuaian |
| :-: | :--- | :--- | :--- | :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| 1 | `PDS-UJI-0001` | Budi Pratama (Mahasiswa) | Dine In (Meja 02) | QRIS (Simulasi) | Americano (Ice) [Rp 16,000] x[2] | Rp 32,000 | Rp 8,000 | **Rp 24,000** | **Rp 24,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 2 | `PDS-UJI-0002` | Siti Aisyah (Mahasiswa) | Takeaway (Meja -) | Tunai di Kasir | Caramel (Hot) [Rp 15,000] x[1] | Rp 15,000 | Rp 0 | **Rp 15,000** | **Rp 15,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 3 | `PDS-UJI-0003` | Ahmad Dani (Mahasiswa) | Dine In (Meja 05) | QRIS (Simulasi) | Mix Platter + French Fries [Rp 20,000 + Rp 12,000] x[1 + 1] | Rp 32,000 | Rp 8,000 | **Rp 24,000** | **Rp 24,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 4 | `PDS-UJI-0004` | Rina Wulandari (Umum) | Dine In (Meja 08) | Tunai di Kasir | Vanilla Latte (Ice) [Rp 18,000] x[2] | Rp 36,000 | Rp 0 | **Rp 36,000** | **Rp 36,000** | Rp 0 | Dibatalkan | **Sesuai (100%)** |
| 5 | `PDS-UJI-0005` | Dimas Setiawan (Umum) | Dine In (Meja 04) | Tunai di Kasir | Mie Rebus Spesial + Telur Rebus/Goreng + Ginger Tea (Hot) [Rp 12,000 + Rp 4,000 + Rp 7,000] x[1 + 1 + 1] | Rp 23,000 | Rp 0 | **Rp 23,000** | **Rp 23,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 6 | `PDS-UJI-0006` | Nurul Hidayah (Mahasiswa) | Takeaway (Meja -) | QRIS (Simulasi) | Matcha Latte + Choco Milk [Rp 18,000 + Rp 18,000] x[1 + 1] | Rp 36,000 | Rp 9,000 | **Rp 27,000** | **Rp 27,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 7 | `PDS-UJI-0007` | Hendra Saputra (Umum) | Dine In (Meja 01) | QRIS (Simulasi) | Brown Sugar (Ice) + Mie Goreng Spesial [Rp 18,000 + Rp 12,000] x[1 + 1] | Rp 30,000 | Rp 0 | **Rp 30,000** | **Rp 30,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 8 | `PDS-UJI-0008` | Fajar Ramadhan (Mahasiswa) | Takeaway (Meja -) | QRIS (Simulasi) | Spanish Latte (Ice) + French Fries [Rp 20,000 + Rp 12,000] x[1 + 1] | Rp 32,000 | Rp 8,000 | **Rp 24,000** | **Rp 24,000** | Rp 0 | Dibatalkan | **Sesuai (100%)** |
| 9 | `PDS-UJI-0009` | Budi Pratama (Mahasiswa) | Dine In (Meja 02) | Tunai di Kasir | Americano (Ice) + Mie Rebus/Goreng Aneka Rasa [Rp 16,000 + Rp 7,000] x[1 + 1] | Rp 23,000 | Rp 0 | **Rp 23,000** | **Rp 23,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 10 | `PDS-UJI-0010` | Wahyu Kurniawan (Umum) | Dine In (Meja 06) | Tunai di Kasir | Espresso + Mix Platter [Rp 10,000 + Rp 20,000] x[1 + 1] | Rp 30,000 | Rp 0 | **Rp 30,000** | **Rp 30,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 11 | `PDS-UJI-0011` | Maya Anggraini (Mahasiswa) | Takeaway (Meja -) | QRIS (Simulasi) | Strawberry Milk [Rp 18,000] x[2] | Rp 36,000 | Rp 9,000 | **Rp 27,000** | **Rp 27,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 12 | `PDS-UJI-0012` | Tri Utami (Umum) | Dine In (Meja 09) | Tunai di Kasir | Caramel (Ice) + Telur Rebus/Goreng [Rp 18,000 + Rp 4,000] x[1 + 1] | Rp 22,000 | Rp 0 | **Rp 22,000** | **Rp 22,000** | Rp 0 | Dibatalkan | **Sesuai (100%)** |
| 13 | `PDS-UJI-0013` | Bagas Wicaksono (Umum) | Takeaway (Meja -) | QRIS (Simulasi) | Lemongrass (Ice) + Mie Rebus Spesial [Rp 12,000 + Rp 12,000] x[1 + 1] | Rp 24,000 | Rp 0 | **Rp 24,000** | **Rp 24,000** | Rp 0 | Selesai | **Sesuai (100%)** |
| 14 | `PDS-UJI-0014` | Cindy Claudia (Mahasiswa) | Dine In (Meja 10) | QRIS (Simulasi) | Vanilla Latte (Hot) + French Fries [Rp 15,000 + Rp 12,000] x[1 + 1] | Rp 27,000 | Rp 6,750 | **Rp 20,250** | **Rp 20,250** | Rp 0 | Selesai | **Sesuai (100%)** |
| 15 | `PDS-UJI-0015` | Reza Pahlevi (Umum) | Dine In (Meja 11) | Tunai di Kasir | Spanish Latte (Hot) + Mix Platter [Rp 18,000 + Rp 20,000] x[1 + 1] | Rp 38,000 | Rp 0 | **Rp 38,000** | **Rp 38,000** | Rp 0 | Selesai | **Sesuai (100%)** |

### Ringkasan Finansial Transaksi Selesai (12 Pesanan Lunas):
* **Total Omzet Perhitungan Manual:** Rp 305,250
* **Total Omzet Pencatatan Sistem:** Rp 305,250
* **Total Potongan Diskon Diberikan:** Rp 40,750
* **Total Selisih Finansial:** **Rp 0 (Akurat 100%)**

---

## 4. Audit dan Rekonsiliasi Stok Akhir

### Rumus Pengujian Stok:
$$\text{Stok Akhir Ekspektasi} = \text{Stok Awal} - \text{Jumlah Terjual (Lunas)} + \text{Jumlah Dikembalikan (Batal)}$$

* Catatan: Pesanan dibatalkan (Order PDS-UJI-0004, PDS-UJI-0008, PDS-UJI-0012) menguji mekanisme *restore stock* otomatis oleh fungsi `CancelOrder()`.

| No | Nama Produk | Stok Awal | Terjual (Lunas) | Batal (Stok Kembali) | Stok Akhir Fisik (Ekspektasi) | Stok Akhir Sistem | Selisih (*Delta*) | Status Audit |
| :-: | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| 1 | **Americano** | 33 | 3 | 0 | 30 | 30 | 0 | **Sesuai** |
| 2 | **Espresso** | 34 | 1 | 0 | 33 | 33 | 0 | **Sesuai** |
| 3 | **Caramel** | 34 | 1 | 1 | 33 | 33 | 0 | **Sesuai** |
| 4 | **Vanilla Latte** | 35 | 1 | 2 | 34 | 34 | 0 | **Sesuai** |
| 5 | **Brown Sugar** | 35 | 1 | 0 | 34 | 34 | 0 | **Sesuai** |
| 6 | **Spanish Latte** | 35 | 1 | 1 | 34 | 34 | 0 | **Sesuai** |
| 7 | **Ginger Tea** | 25 | 1 | 0 | 24 | 24 | 0 | **Sesuai** |
| 8 | **Lemongrass** | 25 | 1 | 0 | 24 | 24 | 0 | **Sesuai** |
| 9 | **Choco Milk** | 25 | 1 | 0 | 24 | 24 | 0 | **Sesuai** |
| 10 | **Strawberry Milk** | 25 | 2 | 0 | 23 | 23 | 0 | **Sesuai** |
| 11 | **Matcha Latte** | 25 | 1 | 0 | 24 | 24 | 0 | **Sesuai** |
| 12 | **Mie Rebus/Goreng Aneka Rasa** | 20 | 1 | 0 | 19 | 19 | 0 | **Sesuai** |
| 13 | **Mie Rebus Spesial** | 20 | 2 | 0 | 18 | 18 | 0 | **Sesuai** |
| 14 | **Mie Goreng Spesial** | 20 | 1 | 0 | 19 | 19 | 0 | **Sesuai** |
| 15 | **Telur Rebus/Goreng** | 20 | 1 | 1 | 19 | 19 | 0 | **Sesuai** |
| 16 | **French Fries** | 24 | 2 | 1 | 22 | 22 | 0 | **Sesuai** |
| 17 | **Mix Platter (Sosis dan Kentang)** | 25 | 3 | 0 | 22 | 22 | 0 | **Sesuai** |

---

## 5. Perhitungan Persentase Kesesuaian (KPI 1–4)

### A. Persentase Kesesuaian Transaksi (Perhitungan Harga & Diskon):
$$\text{Kesesuaian Pesanan} = \frac{\text{Jumlah Pesanan Sesuai}}{\text{Jumlah Pesanan Uji}} \times 100\%$$
$$\text{Kesesuaian Pesanan} = \frac{15}{15} \times 100\% = \mathbf{100.0\%}$$

### B. Persentase Kesesuaian Audit Stok:
$$\text{Kesesuaian Stok} = \frac{\text{Jumlah Produk Sesuai}}{\text{Total Produk Diuji}} \times 100\%$$
$$\text{Kesesuaian Stok} = \frac{17}{17} \times 100\% = \mathbf{100.0\%}$$

---

## 6. Kesimpulan Hasil Pengujian
1. **Perhitungan Finansial Sempurna:** Dari 15 skenario transaksi dengan berbagai kombinasi menu, tipe layanan, dan metode pembayaran, seluruh nilai subtotal, potongan diskon mahasiswa 25% (khusus total belanja di atas Rp 20.000), serta total akhir memiliki **selisih Rp 0** antara hitungan manual dan sistem.
2. **Integritas Stok Teruji:** Pengurangan stok pada transaksi selesai dan pengembalian stok pada 3 transaksi yang dibatalkan berjalan secara presisi tanpa ada stok yang hilang atau menggantung.
3. **Kompatibilitas Excel:** Data mentah telah diekspor ke berkas `uji_transaksi_stok.csv` (UTF-8 BOM) sehingga siap dibuka dan diolah langsung di Microsoft Excel.
