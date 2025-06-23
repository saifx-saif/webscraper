# Adidas Japan Product Scraper

This project contains two Go scripts to scrape product IDs and details from the Adidas Japan website for men's T-shirts, polo shirts, and jackets, saving the data to CSV and Excel files.

## Overview

1. **Extract IDs from HTML**:
   - `scrape_adidas_skus.go` uses a headless browser to fetch HTML from category pages, extracts product IDs (e.g., `HB9386`) using regex, and saves them to `skus_from_html.txt` file where i used `skus.py` script because of some error in golang.
2. **Fetch Product Data via API**:
   - `crawler.go` reads IDs from `skus_from_html.txt`, fetches JSON data from the Adidas API (`https://www.adidas.jp/api/products/{id}`), and saves product details to `adidas_products.csv`.

## Scripts

- **extract_skus_from_html.go**:
  - Scrapes categories: T-shirts, polo shirts, jackets (`https://www.adidas.jp/...`).
  - Pages: `?start=0, 48, 96` (first three pages).
  - Extracts IDs using regex: `href="[^"]*/([A-Z]{2}[0-9]{4})\.html`.
  - Saves raw HTML for debugging.

- **crawler.go**:
  - Reads IDs from `skus.txt`.
  - Fetches data from `https://www.adidas.jp/api/products/{id}`.
  - Saves to CSV  with 17 columns: ID, URL (`https://shop.adidas.jp/products/{id}`), Name, Price, etc.
  - Includes retries, browser-like headers, and gzip support.
  - Logs raw JSON, parsed data, and file sizes.

## Notes

- **API Access**: If 403 errors occur, check `error_403_attempt_*.html`. You may need an API key from `https://adidas.github.io` (add to `crawler.go` headers).
- **Debugging**:
  - Check `response_page_*.html` for HTML content issues.
  - Verify `skus_from_html.txt` has IDs (`wc -l skus.txt`).
  - Monitor console logs for fetch/write errors.
- **Output Verification**:
  - CSV: `head -n 2 adidas_products.csv`.
  - File sizes: `ls -l adidas_products.csv`.

## Example Output

- **skus_from_html.txt**:
  ```
  HB9386
  IA4845
  ...
  ```
- **adidas_products.csv** (sample row):
  ```
  ID,URL,Name,Price,...
  HB9386,https://shop.adidas.jp/products/HB9386,Product Name,5000 JPY,...
  ```

</xArtifact>

