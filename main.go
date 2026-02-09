package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// --- STRUKTUR DATA ---
type ApiResponse struct {
	Meta MetaData         `json:"meta"`
	Data []ArticlePreview `json:"data"`
}

type MetaData struct {
	CurrentPage int    `json:"current_page"`
	TotalPages  int    `json:"total_pages"`
	HasNext     bool   `json:"has_next"`
	HasPrev     bool   `json:"has_prev"`
	NextPageURL string `json:"next_page_url"`
	PrevPageURL string `json:"prev_page_url"`
	LastPageURL string `json:"last_page_url"`
}

type ArticlePreview struct {
	Title      string `json:"title"`
	OriginalID string `json:"original_id"`
	ApiLink    string `json:"api_link"`
	SourceLink string `json:"source_link"`
}

type ArticleDetail struct {
	ID       string              `json:"id"`
	Title    string              `json:"title"`
	Category string              `json:"category"`
	Intro    string              `json:"intro"`
	Products []ProductComparison `json:"products"`
}

// UPDATE: Struktur Data Produk
type ProductComparison struct {
	Rank        int      `json:"rank"`
	BrandName   string   `json:"brand_name"`
	ProductName string   `json:"product_name"`
	Price       string   `json:"price"`
	Images      []string `json:"images"`    // Semua Gambar (Array)
	ImageURL    string   `json:"image_url"` // Gambar Utama (String) - DIPERBAIKI
	Point       string   `json:"point"`
	ShopeeLink  string   `json:"shopee_link"`
}

type HtmlExtraData struct {
	Point      string
	ShopeeLink string
}

// --- STRUKTUR JSON-LD ---
type JsonBreadcrumb struct {
	Type            string `json:"@type"`
	ItemListElement []struct {
		Position int    `json:"position"`
		Name     string `json:"name"`
	} `json:"itemListElement"`
}

type JsonArticle struct {
	Type        string `json:"@type"`
	Headline    string `json:"headline"`
	Description string `json:"description"`
	MainEntity  []struct {
		Type            string `json:"@type"`
		ItemListElement []struct {
			Position int `json:"position"`
			Item     struct {
				Name  string   `json:"name"`
				Image []string `json:"image"`
				Brand struct {
					Name string `json:"name"`
				} `json:"brand"`
				Offers struct {
					LowPrice interface{} `json:"lowPrice"`
				} `json:"offers"`
			} `json:"item"`
		} `json:"itemListElement"`
	} `json:"mainEntity"`
}

// --- HELPER FUNCTIONS ---

func cleanText(s *goquery.Selection) string {
	clone := s.Clone()
	clone.Find("style").Remove()
	clone.Find("script").Remove()
	return strings.TrimSpace(clone.Text())
}

func extractRankInt(s string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(s)
	if val, err := strconv.Atoi(match); err == nil {
		return val
	}
	return 0
}

func formatRupiah(rawPrice interface{}) string {
	if rawPrice == nil {
		return ""
	}
	var priceStr string
	switch v := rawPrice.(type) {
	case string:
		priceStr = v
	case float64:
		priceStr = fmt.Sprintf("%.0f", v)
	case int:
		priceStr = fmt.Sprintf("%d", v)
	default:
		return ""
	}
	re := regexp.MustCompile(`[^0-9]`)
	cleanNum := re.ReplaceAllString(priceStr, "")
	if cleanNum == "" {
		return ""
	}
	num, _ := strconv.Atoi(cleanNum)
	p := messagePrinter(num)
	return "Rp " + p
}

func messagePrinter(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func processShopeeLink(originalLink string) string {
	myAffiliateID := "11379810076"
	parsedURL, err := url.Parse(originalLink)
	if err != nil {
		return ""
	}
	var targetURLStr string
	if strings.Contains(parsedURL.Host, "atid.me") {
		targetURLStr = parsedURL.Query().Get("url")
	} else {
		targetURLStr = originalLink
	}
	if !strings.Contains(strings.ToLower(targetURLStr), "shopee") {
		return ""
	}
	shopeeURL, err := url.Parse(targetURLStr)
	if err != nil {
		return ""
	}
	params := shopeeURL.Query()
	params.Set("affiliate_id", myAffiliateID)
	shopeeURL.RawQuery = params.Encode()
	return shopeeURL.String()
}

// --- ROUTER ---
func mainRouteHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 3 && parts[1] == "detail" {
		handleDetailScrape(w, parts[2])
	} else {
		page := 1
		if len(parts) >= 2 && parts[1] != "" {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				page = p
			}
		}
		handleListScrape(w, page)
	}
}

// --- HANDLER LIST ---
func handleListScrape(w http.ResponseWriter, page int) {
	baseURL := "http://localhost:8080/api"
	response := ApiResponse{
		Meta: MetaData{CurrentPage: page, TotalPages: 0},
		Data: []ArticlePreview{},
	}
	c := colly.NewCollector(
		colly.AllowedDomains("id.my-best.com"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36"),
	)
	seen := make(map[string]bool)
	c.OnHTML("div[data-testid='content_list_item']", func(e *colly.HTMLElement) {
		link := e.ChildAttr("a", "href")
		title := cleanText(e.DOM.Find("h2"))
		parts := strings.Split(link, "/")
		id := parts[len(parts)-1]
		if id != "" && !seen[id] {
			seen[id] = true
			response.Data = append(response.Data, ArticlePreview{
				Title:      title,
				OriginalID: id,
				ApiLink:    fmt.Sprintf("%s/detail/%s", baseURL, id),
				SourceLink: link,
			})
		}
	})
	c.OnHTML("nav[role='navigation'] li", func(e *colly.HTMLElement) {
		text := cleanText(e.DOM)
		if num, err := strconv.Atoi(text); err == nil {
			if num > response.Meta.TotalPages {
				response.Meta.TotalPages = num
			}
		}
	})
	targetURL := "https://id.my-best.com/presses"
	if page > 1 {
		targetURL = fmt.Sprintf("https://id.my-best.com/presses?page=%d", page)
	}
	c.Visit(targetURL)
	if response.Meta.TotalPages > 0 {
		response.Meta.LastPageURL = fmt.Sprintf("%s/%d", baseURL, response.Meta.TotalPages)
		if page < response.Meta.TotalPages {
			response.Meta.HasNext = true
			response.Meta.NextPageURL = fmt.Sprintf("%s/%d", baseURL, page+1)
		}
		if page > 1 {
			response.Meta.HasPrev = true
			if page == 2 {
				response.Meta.PrevPageURL = baseURL
			} else {
				response.Meta.PrevPageURL = fmt.Sprintf("%s/%d", baseURL, page-1)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// --- HANDLER DETAIL ---
func handleDetailScrape(w http.ResponseWriter, id string) {
	detail := ArticleDetail{
		ID:       id,
		Products: []ProductComparison{},
	}
	htmlDataMap := make(map[int]HtmlExtraData)
	c := colly.NewCollector(
		colly.AllowedDomains("id.my-best.com"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36"),
	)

	// 1. PHASE HTML TABLE
	c.OnHTML("table[data-testid='comparison-table'] tbody tr", func(e *colly.HTMLElement) {
		rankText := cleanText(e.DOM.Find("td").Eq(0))
		rank := extractRankInt(rankText)
		if rank > 0 {
			var shopeeLink string
			e.DOM.Find("td").Eq(3).Find("a").Each(func(_ int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if exists {
					processedLink := processShopeeLink(href)
					if processedLink != "" {
						shopeeLink = processedLink
						return
					}
				}
			})
			point := cleanText(e.DOM.Find("td").Eq(4))
			htmlDataMap[rank] = HtmlExtraData{Point: point, ShopeeLink: shopeeLink}
		}
	})

	// 2. PHASE JSON-LD
	c.OnHTML("script[type='application/ld+json']", func(e *colly.HTMLElement) {
		content := e.Text
		if strings.Contains(content, "BreadcrumbList") {
			var bc JsonBreadcrumb
			if err := json.Unmarshal([]byte(content), &bc); err == nil {
				if len(bc.ItemListElement) >= 2 {
					detail.Category = strings.Title(strings.ToLower(bc.ItemListElement[1].Name))
				}
			}
		}
		if strings.Contains(content, "\"Article\"") && strings.Contains(content, "mainEntity") {
			var art JsonArticle
			if err := json.Unmarshal([]byte(content), &art); err == nil {
				detail.Title = art.Headline
				detail.Intro = art.Description
				for _, entity := range art.MainEntity {
					if entity.Type == "ItemList" {
						for _, item := range entity.ItemListElement {
							rank := item.Position
							brandName := item.Item.Brand.Name
							fullName := item.Item.Name
							productName := fullName
							if strings.Contains(fullName, "\n") {
								parts := strings.Split(fullName, "\n")
								if len(parts) > 1 {
									productName = parts[1]
								}
							} else {
								productName = strings.TrimPrefix(productName, brandName)
								productName = strings.TrimSpace(productName)
							}

							// Ambil Gambar Utama untuk field ImageURL
							imgURL := ""
							if len(item.Item.Image) > 0 {
								imgURL = item.Item.Image[0]
							}

							// Ambil Semua Gambar
							images := item.Item.Image

							// Ambil Harga
							rawPrice := item.Item.Offers.LowPrice
							priceFormatted := formatRupiah(rawPrice)

							extras := htmlDataMap[rank]

							prod := ProductComparison{
								Rank:        rank,
								BrandName:   brandName,
								ProductName: productName,
								Price:       priceFormatted,
								Images:      images, // Semua Gambar
								ImageURL:    imgURL, // Gambar Utama (Fix Error disini)
								Point:       extras.Point,
								ShopeeLink:  extras.ShopeeLink,
							}
							detail.Products = append(detail.Products, prod)
						}
					}
				}
			}
		}
	})
	fmt.Printf("Scraping Detail ID %s...\n", id)
	c.Visit(fmt.Sprintf("https://id.my-best.com/%s", id))
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(detail); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	response := map[string]interface{}{
		"status":  "active",
		"message": "MyBest Scraper API is Running!",
		"version": "1.3.0",
		"endpoints": map[string]string{
			"list_articles":  "http://localhost:8080/api",
			"detail_article": "http://localhost:8080/api/detail/{id}",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(response)
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/", mainRouteHandler)
	http.HandleFunc("/api", mainRouteHandler)
	fmt.Println("Server Ready di http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
