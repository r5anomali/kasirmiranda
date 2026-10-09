# KasirMiranda API

Backend REST API untuk aplikasi kasir dan manajemen warung, dibuat menggunakan Go, Gin, GORM, dan MySQL. API menyediakan autentikasi JWT, pengelolaan katalog/stok produk, pencatatan penjualan, ringkasan income/outcome, dan ekspor laporan PDF.

## Fitur

- Autentikasi login menggunakan username/email dan password bcrypt.
- JWT Bearer token untuk melindungi endpoint.
- Logout dengan token blacklist yang disimpan di database.
- Membuat dan melihat produk dengan SKU unik, barcode opsional, kategori, satuan, harga beli/jual, stok, stok minimum, dan status aktif.
- Filter daftar produk berdasarkan kategori dan stok rendah, dengan pagination.
- Mencatat penjualan multi-produk dalam satu transaksi database: stok diperiksa dan dikurangi, serta income dicatat secara atomik.
- Melihat ringkasan income dan outcome berdasarkan periode.
- Memfilter outcome dengan tipe `debt` (hutang pembelian) atau `loan` (pinjaman uang).
- Mengunduh laporan income/outcome dalam PDF dengan format mata uang Rupiah.

## Teknologi

- Go
- Gin
- GORM
- MySQL
- JWT (`github.com/golang-jwt/jwt/v5`)
- bcrypt (`golang.org/x/crypto/bcrypt`)
- gofpdf (`github.com/jung-kurt/gofpdf/v2`)

## Persyaratan

- Go sesuai versi pada `go.mod`.
- MySQL yang dapat diakses dari aplikasi.
- `curl` atau Postman untuk menguji API.

## Menjalankan secara lokal

1. Clone repository dan masuk ke folder project:

   ```bash
   git clone https://github.com/PakpahanLeo/kasirmiranda.git
   cd kasirmiranda
   ```

2. Siapkan MySQL dan buat database:

   ```sql
   CREATE DATABASE kasirmiranda
     CHARACTER SET utf8mb4
     COLLATE utf8mb4_unicode_ci;
   ```

3. Salin `.env.example` menjadi `.env`, kemudian isi konfigurasi lokal:

   ```bash
   cp .env.example .env
   ```

   Contoh konfigurasi:

   ```env
   DB_USER=root
   DB_PASSWORD=your_mysql_password
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_NAME=kasirmiranda
   JWT_SECRET=replace-with-a-random-secret-at-least-32-characters
   PORT=8080
   ```

   `DATABASE_URL` dapat digunakan sebagai alternatif konfigurasi `DB_*` bila menyediakan DSN MySQL dalam format driver Go MySQL. Jangan commit `.env` atau membagikan nilai rahasianya.

4. Unduh dependency dan jalankan API:

   ```bash
   go mod download
   go run .
   ```

   Saat koneksi database berhasil, aplikasi menjalankan GORM `AutoMigrate` untuk model user, product, transaction, dan token blacklist.

5. Periksa server:

   ```bash
   curl http://localhost:8080/health
   ```

   Respons:

   ```json
   {"status":"ok"}
   ```

## Konfigurasi environment

| Variable | Keterangan |
|---|---|
| `DB_USER` | User MySQL lokal |
| `DB_PASSWORD` | Password MySQL lokal |
| `DB_HOST` | Host MySQL lokal |
| `DB_PORT` | Port MySQL lokal, umumnya `3306` |
| `DB_NAME` | Nama database |
| `DATABASE_URL` | DSN MySQL alternatif; jika terisi, menggantikan konfigurasi `DB_*` |
| `JWT_SECRET` | Secret untuk menandatangani JWT; minimal 32 karakter |
| `PORT` | Port HTTP aplikasi; default `8080` |

Di Railway, set environment variables pada service API. Gunakan variabel koneksi database yang disediakan service MySQL Railway dan pastikan format `DATABASE_URL` sesuai dengan driver MySQL Go. Jangan menaruh credential production dalam repository.

## Autentikasi

Login menghasilkan access token. Kirim token tersebut ke endpoint terlindungi menggunakan header:

```http
Authorization: Bearer <access_token>
```

Contoh login lokal:

```bash
curl --location 'http://localhost:8080/auth/login' \
  --header 'Content-Type: application/json' \
  --data '{
    "username": "user@example.com",
    "password": "your-password"
  }'
```

Akun harus sudah tersedia pada tabel `users`, dengan password tersimpan sebagai bcrypt hash. API saat ini belum menyediakan endpoint registrasi.

## Endpoint API

Semua endpoint selain `GET /health` dan `POST /auth/login` membutuhkan Bearer token.

| Method | Path | Keterangan |
|---|---|---|
| `GET` | `/health` | Health check |
| `POST` | `/auth/login` | Login dan memperoleh JWT |
| `GET` | `/auth/me` | Mendapatkan user dari token |
| `POST` | `/auth/logout` | Mencabut token aktif melalui blacklist |
| `POST` | `/products` | Membuat produk |
| `GET` | `/products` | Daftar produk dengan filter dan pagination |
| `POST` | `/sales` | Mencatat penjualan dan mengurangi stok |
| `GET` | `/income` | Ringkasan serta transaksi income |
| `GET` | `/income/export` | Unduh laporan income PDF |
| `GET` | `/outcome` | Ringkasan serta transaksi outcome |
| `GET` | `/outcome/export` | Unduh laporan outcome PDF |

### Products

Buat produk:

```bash
curl --location 'http://localhost:8080/products' \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer <access_token>' \
  --data '{
    "sku": "WRG-0001",
    "barcode": null,
    "name": "Rokok Gudang Garam Surya 12",
    "category": "Rokok",
    "unit": "bungkus",
    "purchasePrice": 20000,
    "sellingPrice": 22000,
    "stock": 15,
    "initialStock": 15,
    "stockMinimum": 5,
    "active": true
  }'
```

`sku` harus unik. `barcode` bersifat opsional dan harus unik jika diisi. `createdAt` dan `updatedAt` opsional, dengan format `YYYY-MM-DD HH:MM:SS`; jika tidak diberikan, waktu aplikasi digunakan.

Daftar produk:

```text
GET /products?category=Rokok&lowStockOnly=true&page=1&limit=20
```

Parameter:

- `category`: filter kategori secara exact match; opsional.
- `lowStockOnly`: `true` untuk mengambil produk dengan `stock <= 5`; opsional.
- `page`: nomor halaman, default `1`.
- `limit`: jumlah item per halaman, default `20`, maksimum `100`.

### Penjualan kasir

`POST /sales` menerima daftar item berdasarkan ID produk. Setiap produk harus aktif dan memiliki stok cukup. Semua perubahan stok dan baris income disimpan dalam satu transaksi database; jika salah satu item gagal, seluruh penjualan dibatalkan.

```bash
curl --location 'http://localhost:8080/sales' \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer <access_token>' \
  --data '{
    "items": [
      {"productId": 14, "quantity": 1},
      {"productId": 15, "quantity": 2}
    ]
  }'
```

Harga penjualan diambil dari `sellingPrice` produk saat transaksi dibuat. Baris income dicatat per item dengan invoice yang sama.

### Income dan outcome

Income:

```text
GET /income?period=today
```

Outcome:

```text
GET /outcome?period=today&type=loan
```

Periode yang didukung: `today`, `week` (7 hari terakhir termasuk hari ini), dan `month` (bulan kalender berjalan). Untuk outcome, parameter `type` opsional dan menerima `debt` atau `loan`.

### Ekspor PDF

```bash
curl --location 'http://localhost:8080/income/export?period=today' \
  --header 'Authorization: Bearer <access_token>' \
  --output income-today.pdf
```

Untuk outcome:

```bash
curl --location 'http://localhost:8080/outcome/export?period=today&type=debt' \
  --header 'Authorization: Bearer <access_token>' \
  --output outcome-debt-today.pdf
```

Nominal dalam PDF ditampilkan dalam format Rupiah, misalnya `Rp.22.000`.

## Model data

- `users`: identitas login, bcrypt password hash, role.
- `products`: SKU/barcode, identitas produk, kategori/satuan, harga, stok, status.
- `transactions`: income/outcome. Penjualan menghasilkan baris `type=income`; outcome membedakan `sub_type=debt` atau `sub_type=loan`.
- `token_blacklists`: hash signature token yang telah logout dan waktu kedaluwarsanya.

## Menjalankan pemeriksaan

```bash
go test ./...
```

## Catatan deployment Railway

1. Buat service MySQL dan service API dari repository ini.
2. Atur environment variables koneksi database, `JWT_SECRET` yang kuat, dan `PORT` sesuai konfigurasi Railway.
3. Pastikan service API dapat mengakses database melalui jaringan internal Railway.
4. Deploy aplikasi. Migrasi tabel dijalankan otomatis saat startup apabila koneksi DB berhasil.
5. Pasang custom domain dan konfigurasi DNS/TLS bila menggunakan domain sendiri.

Untuk production, gunakan secret acak yang kuat, jangan memakai kredensial contoh, dan jangan commit file `.env`.
