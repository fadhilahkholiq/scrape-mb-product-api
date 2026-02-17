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

type ApiResponse struct {
	Meta MetaData         `json:"meta"`
	Data []ArticlePreview `json:"data"`
}

type CategoryResponse struct {
	Meta MetaData   `json:"meta"`
	Data []Category `json:"data"`
}

type Category struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Slug     string `json:"slug"`
	ApiLink  string `json:"api_link"`
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
}

type ArticleDetail struct {
	ID       string              `json:"id"`
	Title    string              `json:"title"`
	Category string              `json:"category"`
	Products []ProductComparison `json:"products"`
}

type ProductComparison struct {
	Rank        int      `json:"rank"`
	BrandName   string   `json:"brand_name"`
	ProductName string   `json:"product_name"`
	Price       string   `json:"price"`
	Images      []string `json:"images"`
	ImageURL    string   `json:"image_url"`
	Point       string   `json:"point"`
	ShopeeLink  string   `json:"shopee_link"`
}

type HtmlExtraData struct {
	Point      string
	ShopeeLink string
}

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

func mainRouteHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) >= 3 && parts[1] == "detail" {
		handleDetailScrape(w, parts[2])
		return
	}

	if len(parts) >= 2 && parts[1] == "categories" {
		if len(parts) >= 3 && parts[2] != "" {
			page := 1
			if p := r.URL.Query().Get("page"); p != "" {
				if val, err := strconv.Atoi(p); err == nil {
					page = val
				}
			}
			handleCategoryArticles(w, parts[2], page)
		} else {
			handleCategoryList(w)
		}
		return
	}

	page := 1
	if len(parts) >= 2 && parts[1] != "" {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			page = p
		}
	}
	handleListScrape(w, page)
}

func handleListScrape(w http.ResponseWriter, page int) {
	baseURL := "/api"
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
	encoder.Encode(response)
}

func handleCategoryList(w http.ResponseWriter) {
	response := CategoryResponse{
		Meta: MetaData{CurrentPage: 1, TotalPages: 1},
		Data: []Category{},
	}

	c := colly.NewCollector(
		colly.AllowedDomains("id.my-best.com"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36"),
	)

	seen := make(map[string]bool)

	c.OnHTML("li a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		if strings.Contains(link, "/categories/") || strings.Contains(link, "/tags/") {
			parts := strings.Split(link, "/")
			if len(parts) > 0 {
				slug := parts[len(parts)-1]

				if slug != "" && !seen[slug] && len(slug) > 0 {
					if _, err := strconv.Atoi(slug); err == nil {
						seen[slug] = true

						imgURL := ""
						e.DOM.Find("noscript").Each(func(_ int, s *goquery.Selection) {
							htmlContent := s.Text()
							re := regexp.MustCompile(`src="([^"]+)"`)
							match := re.FindStringSubmatch(htmlContent)
							if len(match) > 1 && strings.Contains(match[1], "img.id.my-best.com") {
								imgURL = match[1]
							}
						})

						if imgURL == "" {
							e.DOM.Find("img").Each(func(_ int, s *goquery.Selection) {
								src, _ := s.Attr("src")
								if strings.Contains(src, "img.id.my-best.com") {
									imgURL = src
								}
							})
						}

						name := ""
						e.DOM.Find("img").First().Parent().Children().Each(func(_ int, s *goquery.Selection) {
							if s.Is("div") && s.Find("img").Length() == 0 && s.Find("noscript").Length() == 0 {
								titleDiv := s.Children().First()
								if titleDiv.Length() > 0 {
									titleClone := titleDiv.Clone()
									titleClone.Find("style, script, svg").Remove()
									name = strings.TrimSpace(titleClone.Text())
								}
							}
						})

						if name == "" || strings.Contains(name, "{") {
							clone := e.DOM.Clone()
							clone.Find("img, noscript, svg, iconify-icon, style, script").Remove()
							clone.Find("div, span, p").BeforeHtml("|||")

							fullText := clone.Text()
							parts := strings.Split(fullText, "|||")

							for _, part := range parts {
								cleaned := strings.TrimSpace(part)
								if len(cleaned) >= 2 && !strings.HasPrefix(cleaned, ".") && !strings.Contains(cleaned, "{") {
									name = cleaned
									break
								}
							}
						}

						if name == "" {
							name = fmt.Sprintf("Category %s", slug)
						}

						name = strings.TrimSpace(name)
						name = strings.Title(strings.ToLower(name))

						response.Data = append(response.Data, Category{
							Name:     name,
							ImageURL: imgURL,
							Slug:     slug,
							ApiLink:  fmt.Sprintf("/api/categories/%s", slug),
						})
					}
				}
			}
		}
	})

	c.Visit("https://id.my-best.com/")
	c.Visit("https://id.my-best.com/presses")

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(response)
}

func handleCategoryArticles(w http.ResponseWriter, categorySlug string, page int) {
	baseURL := fmt.Sprintf("/api/categories/%s", categorySlug)
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
				ApiLink:    fmt.Sprintf("/api/detail/%s", id),
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

	targetURL := fmt.Sprintf("https://id.my-best.com/presses?category=%s", categorySlug)
	if page > 1 {
		targetURL = fmt.Sprintf("%s&page=%d", targetURL, page)
	}

	c.Visit(targetURL)

	if response.Meta.TotalPages > 0 {
		response.Meta.LastPageURL = fmt.Sprintf("%s?page=%d", baseURL, response.Meta.TotalPages)
		if page < response.Meta.TotalPages {
			response.Meta.HasNext = true
			response.Meta.NextPageURL = fmt.Sprintf("%s?page=%d", baseURL, page+1)
		}
		if page > 1 {
			response.Meta.HasPrev = true
			response.Meta.PrevPageURL = fmt.Sprintf("%s?page=%d", baseURL, page-1)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(response)
}

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

							imgURL := ""
							if len(item.Item.Image) > 0 {
								imgURL = item.Item.Image[0]
							}
							images := item.Item.Image
							rawPrice := item.Item.Offers.LowPrice
							priceFormatted := formatRupiah(rawPrice)
							extras := htmlDataMap[rank]

							prod := ProductComparison{
								Rank:        rank,
								BrandName:   brandName,
								ProductName: productName,
								Price:       priceFormatted,
								Images:      images,
								ImageURL:    imgURL,
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
	fmt.Printf("Scraping detail ID %s\n", id)
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
		"message": "API is running!",
		"version": "2.3.0",
		"author":  "KF",
		"endpoints": map[string]string{
			"list_articles":   "/api",
			"list_categories": "/api/categories",
			"detail_category": "/api/categories/{slug}",
			"detail_article":  "/api/detail/{id}",
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
	fmt.Println("Server running!")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
