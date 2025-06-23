package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func loadExistingSKUs(filename string) (map[string]bool, error) {
	skuMap := make(map[string]bool)
	file, err := os.Open(filename)
	if os.IsNotExist(err) {
		return skuMap, nil 
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		sku := strings.TrimSpace(scanner.Text())
		if sku != "" {
			skuMap[sku] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", filename, err)
	}
	return skuMap, nil
}

func extractSKUsFromHTML(filePath string, existingSKUs map[string]bool) ([]string, error) {
	htmlContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
	}

	bodyStr := string(htmlContent)
	var skus []string
	skuMap := make(map[string]bool)

	linkRe := regexp.MustCompile(`href="[^"]*/products/([A-Z]{2}[0-9]{4})[^"]*"`)
	linkMatches := linkRe.FindAllStringSubmatch(bodyStr, -1)
	fmt.Printf("Found %d href matches for /products/[SKU]\n", len(linkMatches))
	for _, match := range linkMatches {
		if len(match) > 1 {
			sku := match[1]
			if !skuMap[sku] && !existingSKUs[sku] {
				skus = append(skus, sku)
				skuMap[sku] = true
				fmt.Printf("Extracted SKU from href: %s\n", sku)
			}
		}
	}

	textRe := regexp.MustCompile(`\b([A-Z]{2}[0-9]{4})\b`)
	textMatches := textRe.FindAllStringSubmatch(bodyStr, -1)
	fmt.Printf("Found %d text matches for SKU pattern\n", len(textMatches))
	for _, match := range textMatches {
		if len(match) > 1 {
			sku := match[1]
			if !skuMap[sku] && !existingSKUs[sku] {
				skus = append(skus, sku)
				skuMap[sku] = true
				fmt.Printf("Extracted SKU from text: %s\n", sku)
			}
		}
	}

	if len(skus) == 0 {
		fmt.Printf("No SKUs extracted from %s\n", filePath)
	}
	return skus, nil
}

func appendSKUs(skus []string, filename string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	for _, sku := range skus {
		if _, err := fmt.Fprintf(file, "%s\n", sku); err != nil {
			return fmt.Errorf("failed to write SKU %s: %v", sku, err)
		}
	}
	return nil
}

func main() {
	fmt.Println("Starting SKU extractor for HTML file...")

	htmlFile := "response_page_1750670937220652501.html"

	existingSKUs, err := loadExistingSKUs("skus_from_html.txt")
	if err != nil {
		fmt.Printf("Failed to load existing SKUs: %v\n", err)
		return
	}
	fmt.Printf("Loaded %d existing SKUs from skus_from_html.txt\n", len(existingSKUs))

	skus, err := extractSKUsFromHTML(htmlFile, existingSKUs)
	if err != nil {
		fmt.Printf("Failed to extract SKUs: %v\n", err)
		return
	}

	fmt.Printf("Extracted %d unique SKUs\n", len(skus))

	fmt.Println("Appending SKUs to skus_from_html.txt...")
	if err := appendSKUs(skus, "skus_from_html.txt"); err != nil {
		fmt.Printf("Failed to save SKUs: %v\n", err)
		return
	}
	fmt.Printf("Successfully appended %d\n", len(skus))
}
