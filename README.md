# 📦 mybest Scraper API

REST API berbasis Golang untuk melakukan scraping daftar rekomendasi produk dari **mybest**, kemudian menyajikannya dalam format JSON.

API ini mendukung:

- List artikel.
- List kategori.
- Artikel berdasarkan kategori.
- Detail berupa ranking produk, gambar, harga, dan link Shopee.
- Pagination.

---

# 🚀 Base URL

## Development

```
http://localhost:8080
```

## Production

```
https://yourdomain.com
```

---

# 🧭 Endpoints Overview

| Endpoint | Method | Deskripsi |
|-----------|--------|------------|
| `/` | GET | Status API |
| `/api` | GET | List artikel (halaman 1) |
| `/api/{page}` | GET | List artikel berdasarkan halaman |
| `/api/categories` | GET | List semua kategori |
| `/api/categories/{slug}` | GET | Artikel berdasarkan kategori |
| `/api/detail/{id}` | GET | Detail artikel |

---

# 🏠 1. Status API

## Endpoint

```
GET /
```

## Response

```json
{
  "status": "active",
  "message": "API is running!",
  "version": "2.3.0",
  "author": "KF",
  "endpoints": {
    "list_articles": "/api",
    "list_categories": "/api/categories",
    "detail_category": "/api/categories/{slug}",
    "detail_article": "/api/detail/{id}"
  }
}
```

---

# 📄 2. List Artikel

Mengambil daftar artikel dari halaman.

## Endpoint

```
GET /api
GET /api/{page}
```

## Contoh Request

```
GET /api/2
```

## Contoh Response

```json
{
  "meta": {
    "current_page": 2,
    "total_pages": 10,
    "has_next": true,
    "has_prev": true,
    "next_page_url": "/api/3",
    "prev_page_url": "/api/1",
    "last_page_url": "/api/10"
  },
  "data": [
    {
      "title": "Skincare Terbaik",
      "original_id": "12345",
      "api_link": "/api/detail/12345"
    }
  ]
}
```

## Meta Fields

| Field | Tipe | Keterangan |
|--------|------|------------|
| current_page | int | Halaman saat ini |
| total_pages | int | Total halaman tersedia |
| has_next | bool | Ada halaman berikutnya |
| has_prev | bool | Ada halaman sebelumnya |
| next_page_url | string | URL halaman berikutnya |
| prev_page_url | string | URL halaman sebelumnya |
| last_page_url | string | URL halaman terakhir |

## Data Fields

| Field | Tipe | Keterangan |
|--------|------|------------|
| title | string | Judul artikel |
| original_id | string | ID artikel dari website asli |
| api_link | string | Endpoint detail artikel |

---

# 🗂 3. List Kategori

Mengambil semua kategori yang tersedia.

## Endpoint

```
GET /api/categories
```

## Contoh Response

```json
{
  "meta": {
    "current_page": 1,
    "total_pages": 1
  },
  "data": [
    {
      "name": "Skincare",
      "image_url": "https://img.id.my-best.com/...",
      "slug": "123",
      "api_link": "/api/categories/123"
    }
  ]
}
```

## Fields

| Field | Tipe | Keterangan |
|--------|------|------------|
| name | string | Nama kategori |
| image_url | string | URL gambar kategori |
| slug | string | ID kategori |
| api_link | string | Endpoint artikel kategori |

---

# 📚 4. Artikel Berdasarkan Kategori

Mengambil artikel berdasarkan kategori tertentu.

## Endpoint

```
GET /api/categories/{slug}
```

## Query Parameter

| Parameter | Tipe | Default | Keterangan |
|------------|------|---------|------------|
| page | int | 1 | Nomor halaman |

## Contoh Request

```
GET /api/categories/123?page=2
```

## Contoh Response

```json
{
  "meta": {
    "current_page": 2,
    "total_pages": 5,
    "has_next": true,
    "has_prev": true,
    "next_page_url": "/api/categories/123?page=3",
    "prev_page_url": "/api/categories/123?page=1"
  },
  "data": [
    {
      "title": "Sunscreen Lokal",
      "original_id": "56789",
      "api_link": "/api/detail/56789"
    }
  ]
}
```

---

# 🔍 5. Detail Artikel

Mengambil detail artikel beserta ranking produk.

## Endpoint

```
GET /api/detail/{id}
```

## Contoh Request

```
GET /api/detail/12345
```

## Contoh Response

```json
{
  "id": "12345",
  "title": "10 Rekomendasi Serum Terbaik",
  "category": "Skincare",
  "products": [
    {
      "rank": 1,
      "brand_name": "Brand A",
      "product_name": "Serum Vitamin C",
      "price": "Rp 125.000",
      "images": [
        "https://img.id.my-best.com/image1.jpg",
        "https://img.id.my-best.com/image2.jpg"
      ],
      "image_url": "https://img.id.my-best.com/image1.jpg",
      "point": "Mengandung vitamin C stabil dan cocok untuk kulit sensitif",
      "shopee_link": "https://shopee.co.id/..."
    }
  ]
}
```

## Root Fields

| Field | Tipe | Keterangan |
|--------|------|------------|
| id | string | ID artikel |
| title | string | Judul artikel |
| category | string | Nama kategori |
| products | array | Daftar produk ranking |

## Product Fields

| Field | Tipe | Keterangan |
|--------|------|------------|
| rank | int | Ranking produk |
| brand_name | string | Nama brand |
| product_name | string | Nama produk |
| price | string | Harga format Rupiah |
| images | array | Semua gambar produk |
| image_url | string | Gambar utama |
| point | string | Keunggulan produk |
| shopee_link | string | Link Shopee |

---

# 🛠 Cara Menjalankan Project

Pastikan Go sudah ter-install.

## Install Dependency

```bash
go mod tidy
```

## Jalankan Server

```bash
go run main.go
```

Server akan berjalan di:

```
http://localhost:8080
```

---

# 📌 Contoh Penggunaan dengan cURL

## List Artikel

```bash
curl http://localhost:8080/api
```

## List Artikel Halaman 2

```bash
curl http://localhost:8080/api/2
```

## List Kategori

```bash
curl http://localhost:8080/api/categories
```

## Detail Artikel

```bash
curl http://localhost:8080/api/detail/12345
```

---

# ⚠️ Catatan Penting

- Dibuat hanya untuk tujuan pembelajaran.
- API ini berbasis scraping.
- Jika struktur web sumber berubah, API bisa terdampak.
- Tidak ada rate limiting bawaan.
- Tidak ada authentication.
- Disarankan menggunakan caching untuk production.
- Disarankan deploy di belakang reverse proxy.
