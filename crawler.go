package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ProductData structure 
type ProductData struct {
	ID             string   `json:"id"`
	URL            string   `json:"url"`
	Name           string   `json:"name"`
	Price          string   `json:"price"`
	Description    string   `json:"description"`
	Images         []string `json:"images"`
	Sizes          []string `json:"sizes"`
	Colors         []string `json:"colors"`
	Availability   string   `json:"availability"`
	Brand          string   `json:"brand"`
	Category       string   `json:"category"`
	Features       []string `json:"features"`
	RatingFitting  string   `json:"rating_fitting"`
	RatingLength   string   `json:"rating_length"`
	RatingQuality  string   `json:"rating_quality"`
	RatingComfort  string   `json:"rating_comfort"`
	AverageRating  string   `json:"average_rating"`
	ReviewCount    string   `json:"review_count"`
}

type ScrapingSession struct {
	client     *http.Client
	baseURL    string
	userAgents []string
}

func NewScrapingSession() *ScrapingSession {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
	userAgents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
	}
	return &ScrapingSession{
		client:     client,
		baseURL:    "https://www.adidas.jp",
		userAgents: userAgents,
	}
}

func (s *ScrapingSession) getRandomUserAgent() string {
	return s.userAgents[rand.Intn(len(s.userAgents))]
}

func (s *ScrapingSession) setCommonHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", s.getRandomUserAgent())
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Referer", s.baseURL+"/s/men/")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="129", "Not=A?Brand";v="8", "Chromium";v="129"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
}

func (s *ScrapingSession) makeRequest(targetURL string, retries int) ([]byte, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	targetURL = parsedURL.String()

	for attempt := 1; attempt <= retries; attempt++ {
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return nil, err
		}
		s.setCommonHeaders(req)

		resp, err := s.client.Do(req)
		if err != nil {
			fmt.Printf("Attempt %d failed: %v\n", attempt, err)
			if attempt < retries {
				time.Sleep(time.Duration(3+rand.Intn(3)) * time.Second)
				continue
			}
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Attempt %d: Status OK\n", attempt)
			body, err := s.readResponseBody(resp)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %v", err)
			}
			return body, nil
		}

		fmt.Printf("Attempt %d: Status %d\n", attempt, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		errorFile := fmt.Sprintf("error_%d_attempt_%d.html", resp.StatusCode, attempt)
		if err := os.WriteFile(errorFile, body, 0644); err != nil {
			fmt.Printf("Failed to save error file %s: %v\n", errorFile, err)
		} else {
			fmt.Printf("Saved error response to %s\n", errorFile)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			fmt.Println("Rate limit hit, waiting longer...")
			time.Sleep(time.Duration(10+rand.Intn(5)) * time.Second)
			continue
		} else if resp.StatusCode == http.StatusForbidden {
			fmt.Printf("403 Forbidden: Check %s for details\n", errorFile)
			time.Sleep(time.Duration(5+rand.Intn(5)) * time.Second)
			continue
		}

		return nil, fmt.Errorf("failed with status: %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("all %d attempts failed", retries)
}

func (s *ScrapingSession) readResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	return body, nil
}

func readSKUs(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read IDs file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var ids []string
	for _, line := range lines {
		id := strings.TrimSpace(line)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *ScrapingSession) getProductDetails(id string) (*ProductData, error) {
	apiURL := fmt.Sprintf("%s/api/products/%s", s.baseURL, id)
	body, err := s.makeRequest(apiURL, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product %s: %v", id, err)
	}

	fmt.Printf("Raw JSON response for ID %s:\n%s\n", id, string(body))

	var data struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		MetaData    struct {
			Description string `json:"description"`
		} `json:"meta_data"`
		ProductListingAssets []struct {
			ImageURL string `json:"image_url"`
		} `json:"product_listing_assets"`
		AttributeList struct {
			Brand        string   `json:"brand"`
			Color        string   `json:"color"`
			Category     string   `json:"category"`
			Functions    []string `json:"functions"`
			ProductFit   []string `json:"productfit"`
			BaseMaterial []string `json:"base_material"`
			IsOrderable  bool     `json:"is_orderable"`
		} `json:"attribute_list"`
		PricingInformation struct {
			CurrentPrice float64 `json:"currentPrice"`
		} `json:"pricing_information"`
		ProductDescription struct {
			Text string   `json:"text"`
			Usps []string `json:"usps"`
		} `json:"product_description"`
		VariationList []struct {
			SKU  string `json:"sku"`
			Size string `json:"size"`
		} `json:"variation_list"`
		ProductLinkList []struct {
			SearchColor  string `json:"search_color"`
			DefaultColor string `json:"default_color"`
		} `json:"product_link_list"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON for %s: %v", id, err)
	}

	product := &ProductData{
		ID:            data.ID,
		URL:           fmt.Sprintf("https://shop.adidas.jp/products/%s", data.ID),
		Name:          data.Name,
		Price:         fmt.Sprintf("%.0f JPY", data.PricingInformation.CurrentPrice),
		Description:   data.ProductDescription.Text,
		Availability:  "Out of Stock",
		Brand:         "Adidas",
		Category:      data.AttributeList.Category,
		RatingFitting: "N/A",
		RatingLength:  "N/A",
		RatingQuality: "N/A",
		RatingComfort: "N/A",
		AverageRating: "N/A",
		ReviewCount:   "N/A",
	}

	if data.AttributeList.IsOrderable {
		product.Availability = "In Stock"
	}

	for _, asset := range data.ProductListingAssets {
		product.Images = append(product.Images, asset.ImageURL)
	}

	for _, variation := range data.VariationList {
		product.Sizes = append(product.Sizes, variation.Size)
	}

	product.Colors = append(product.Colors, data.AttributeList.Color)
	for _, link := range data.ProductLinkList {
		if link.SearchColor != data.AttributeList.Color && link.SearchColor != "" {
			product.Colors = append(product.Colors, link.SearchColor)
		}
	}

	product.Features = append(product.Features, data.AttributeList.Functions...)
	product.Features = append(product.Features, data.AttributeList.ProductFit...)
	product.Features = append(product.Features, data.AttributeList.BaseMaterial...)
	product.Features = append(product.Features, data.ProductDescription.Usps...)

	featureMap := make(map[string]bool)
	var uniqueFeatures []string
	for _, f := range product.Features {
		if !featureMap[f] {
			featureMap[f] = true
			uniqueFeatures = append(uniqueFeatures, f)
		}
	}
	product.Features = uniqueFeatures

	fmt.Printf("  Parsed ProductData for ID %s:\n", id)
	fmt.Printf("  ID: %s\n", product.ID)
	fmt.Printf("  URL: %s\n", product.URL)
	fmt.Printf("  Name: %s\n", product.Name)
	fmt.Printf("  Price: %s\n", product.Price)
	fmt.Printf("  Category: %s\n", product.Category)
	fmt.Printf("  Sizes: %s\n", strings.Join(product.Sizes, ","))
	fmt.Printf("  Colors: %s\n", strings.Join(product.Colors, ","))
	fmt.Printf("  Availability: %s\n", product.Availability)
	fmt.Printf("  Description: %s\n", product.Description)
	fmt.Printf("  Images: %s\n", strings.Join(product.Images, ","))
	fmt.Printf("  Features: %s\n", strings.Join(product.Features, ","))
	fmt.Printf("  Sense of Fitting Rating: %s\n", product.RatingFitting)
	fmt.Printf("  Length Appropriation Rating: %s\n", product.RatingLength)
	fmt.Printf("  Material Quality Rating: %s\n", product.RatingQuality)
	fmt.Printf("  Comfort Rating: %s\n", product.RatingComfort)
	fmt.Printf("  Average Rating: %s\n", product.AverageRating)
	fmt.Printf("  Review Count: %s\n", product.ReviewCount)

	return product, nil
}

func initExcel(filename string) (*excelize.File, int, error) {
	var f *excelize.File
	sheet := "Products"

	if _, err := os.Stat(filename); err == nil {
		f, err = excelize.OpenFile(filename)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open existing Excel file %s: %v", filename, err)
		}
		fmt.Printf("Opened existing Excel file: %s\n", filename)
	} else {
		f = excelize.NewFile()
		index, err := f.NewSheet(sheet)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create new sheet: %v", err)
		}
		f.SetActiveSheet(index)

		headers := []string{
			"ID", "URL", "Name", "Price", "Category", "Sizes", "Colors", "Availability",
			"Description", "Images", "Features", "Sense of Fitting Rating",
			"Length Appropriation Rating", "Material Quality Rating", "Comfort Rating",
			"Average Rating", "Review Count",
		}
		for col, header := range headers {
			cell := fmt.Sprintf("%s1", string(rune('A'+col)))
			f.SetCellValue(sheet, cell, header)
			style, err := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
			})
			if err != nil {
				return nil, 0, fmt.Errorf("failed to create header style: %v", err)
			}
			f.SetCellStyle(sheet, cell, cell, style)
		}

		for col := 'A'; col <= 'Q'; col++ {
			colName := string(col)
			f.SetColWidth(sheet, colName, colName, 20)
		}

		fmt.Printf("Created new Excel file: %s\n", filename)
	}

	if err := f.SaveAs(filename); err != nil {
		return nil, 0, fmt.Errorf("failed to save initial Excel file %s: %v", filename, err)
	}

	if stat, err := os.Stat(filename); err == nil {
		fmt.Printf("Initial Excel file size: %d bytes\n", stat.Size())
	}

	return f, f.GetActiveSheetIndex(), nil
}

func initCSV(filename string) (*os.File, *csv.Writer, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open CSV file %s: %v", filename, err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to stat CSV file %s: %v", filename, err)
	}
	writer := csv.NewWriter(file)

	if stat.Size() == 0 {
		headers := []string{
			"ID", "URL", "Name", "Price", "Category", "Sizes", "Colors", "Availability",
			"Description", "Images", "Features", "Sense of Fitting Rating",
			"",
			"Material Quality Rating", "Comfort Rating", "Average Rating", "Review Count",
		}
		if err := writer.Write(headers); err != nil {
			file.Close()
			return nil, nil, fmt.Errorf("failed to write CSV headers: %v", err)
		}
		writer.Flush()
		fmt.Printf("Created new CSV file with headers: %s\n", filename)
	} else {
		fmt.Printf("Found existing CSV file: %s\n", filename)
	}

	fmt.Printf("Initial CSV file size: %d bytes\n", stat.Size())

	return file, writer, nil
}

func writeProductToExcel(f *excelize.File, sheet string, row int, p *ProductData, filename string) error {
	fmt.Printf("Writing ID %s to Excel at row %d\n", p.ID, row)

	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), p.ID)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.URL)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", row), p.Name)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), p.Price)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), p.Category)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), strings.Join(p.Sizes, ","))
	f.SetCellValue(sheet, fmt.Sprintf("G%d", row), strings.Join(p.Colors, ","))
	f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.Availability)
	f.SetCellValue(sheet, fmt.Sprintf("I%d", row), p.Description)
	f.SetCellValue(sheet, fmt.Sprintf("J%d", row), strings.Join(p.Images, ","))
	f.SetCellValue(sheet, fmt.Sprintf("K%d", row), strings.Join(p.Features, ","))
	f.SetCellValue(sheet, fmt.Sprintf("L%d", row), p.RatingFitting)
	f.SetCellValue(sheet, fmt.Sprintf("M%d", row), p.RatingLength)
	f.SetCellValue(sheet, fmt.Sprintf("N%d", row), p.RatingQuality)
	f.SetCellValue(sheet, fmt.Sprintf("O%d", row), p.RatingComfort)
	f.SetCellValue(sheet, fmt.Sprintf("P%d", row), p.AverageRating)
	f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), p.ReviewCount)

	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save Excel file for row %d (ID %s): %v", row, p.ID, err)
	}

	if stat, err := os.Stat(filename); err == nil {
		fmt.Printf("Excel file size after writing ID %s: %d bytes\n", p.ID, stat.Size())
	} else {
		fmt.Printf(" Failed to get Excel file size: %v\n", err)
	}

	fmt.Printf("Successfully wrote ID %s to Excel at row %d\n", p.ID, row)
	return nil
}

func writeProductToCSV(w *csv.Writer, p *ProductData, filename string) error {
	fmt.Printf("Writing ID %s to CSV\n", p.ID)

	record := []string{
		p.ID,
		p.URL,
		p.Name,
		p.Price,
		p.Category,
		strings.Join(p.Sizes, ","),
		strings.Join(p.Colors, ","),
		p.Availability,
		p.Description,
		strings.Join(p.Images, ","),
		strings.Join(p.Features, ","),
		p.RatingFitting,
		p.RatingLength,
		p.RatingQuality,
		p.RatingComfort,
		p.AverageRating,
		p.ReviewCount,
	}

	if err := w.Write(record); err != nil {
		return fmt.Errorf("failed to write CSV for ID %s: %v", p.ID, err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV for ID %s: %v", p.ID, err)
	}

	if stat, err := os.Stat(filename); err == nil {
		fmt.Printf("CSV file size after writing ID %s: %d bytes\n", p.ID, stat.Size())
	} else {
		fmt.Printf("Failed to get CSV file size: %v\n", err)
	}

	fmt.Printf("Successfully wrote ID %s to CSV\n", p.ID)
	return nil
}

func getNextExcelRow(f *excelize.File, sheet string) int {
	rows, err := f.GetRows(sheet)
	if err != nil {
		fmt.Printf("Failed to get rows for sheet %s, assuming row 1: %v\n", sheet, err)
		return 1
	}
	return len(rows) + 1
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Starting Adidas API crawler for 224 products...")

	fmt.Println("Reading IDs from skus_from_html.txt...")
	ids, err := readSKUs("skus_from_html.txt")
	if err != nil {
		log.Fatalf("Failed to read IDs: %v", err)
	}
	fmt.Printf("Loaded %d IDs\n", len(ids))

	excelFilename := "adidas_products.xlsx"
	f, _, err := initExcel(excelFilename)
	if err != nil {
		log.Fatalf("Failed to initialize Excel file: %v", err)
	}
	defer func() {
		if err := f.SaveAs(excelFilename); err != nil {
			fmt.Printf("Failed to perform final save of Excel file: %v\n", err)
		}
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to close Excel file: %v\n", err)
		}
		fmt.Printf("Closed Excel file: %s\n", excelFilename)
		if stat, err := os.Stat(excelFilename); err == nil {
			fmt.Printf("Final Excel file size: %d bytes\n", stat.Size())
		}
	}()

	csvFilename := "adidas_products.csv"
	csvFile, csvWriter, err := initCSV(csvFilename)
	if err != nil {
		log.Fatalf("Failed to initialize CSV file: %v", err)
	}
	defer func() {
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			fmt.Printf("Failed to flush CSV writer: %v\n", err)
		}
		if err := csvFile.Close(); err != nil {
			fmt.Printf("Failed to close CSV file: %v\n", err)
		}
		fmt.Printf("Closed CSV file: %s\n", csvFilename)
		if stat, err := os.Stat(csvFilename); err == nil {
			fmt.Printf("Final CSV file size: %d bytes\n", stat.Size())
		}
	}()

	session := NewScrapingSession()

	for i, id := range ids {
		fmt.Printf("Fetching ID %s (%d/%d)\n", id, i+1, len(ids))
		product, err := session.getProductDetails(id)
		if err != nil {
			fmt.Printf("Failed to fetch ID %s: %v\n", id, err)
			continue
		}

		row := getNextExcelRow(f, "Products")

		if err := writeProductToExcel(f, "Products", row, product, excelFilename); err != nil {
			fmt.Printf("Failed to write ID %s to Excel: %v\n", id, err)
			continue
		}

		if err := writeProductToCSV(csvWriter, product, csvFilename); err != nil {
			fmt.Printf("Failed to write ID %s to CSV: %v\n", id, err)
			continue
		}

		time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
	}

}
