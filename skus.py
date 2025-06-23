import re
import time
import random
import os
from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

def scrape_category(url, driver):
    all_links = []
    all_skus = set()
    pages = [0, 48, 96] 

    for start in pages:
        page_url = f"{url}?start={start}" if start > 0 else url
        print(f"Scraping page: {page_url}")

        try:
            driver.get(page_url)
            WebDriverWait(driver, 10).until(EC.presence_of_element_located((By.TAG_NAME, "body")))
            driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
            time.sleep(3) 

            html_content = driver.page_source
            html_file = f"response_page_{int(time.time() * 1000)}.html"
            with open(html_file, "w", encoding="utf-8") as f:
                f.write(html_content)
            print(f"Saved raw HTML to {html_file}")

            link_elements = driver.find_elements(By.TAG_NAME, "a")
            page_links = []
            for elem in link_elements:
                href = elem.get_attribute("href")
                if href:
                    page_links.append(href)

            print(f"Found {len(page_links)} <a> links on page")

            sku_pattern = re.compile(r'href="[^"]*/([A-Z]{2}[0-9]{4})\.html')
            page_skus = set()
            for link in page_links:
                matches = sku_pattern.finditer(link)
                for match in matches:
                    sku = match.group(1)
                    page_skus.add(sku)
                    print(f"Matched SKU: {sku} from URL: {link}")

            print(f"Found {len(page_skus)} SKUs on page: {page_skus}")
            all_links.extend(page_links)
            all_skus.update(page_skus)

        except Exception as e:
            print(f"Error scraping {page_url}: {e}")
            continue

    print(f"Total unique SKUs for category: {len(all_skus)}")
    return all_links, list(all_skus)

def save_links_and_skus(categories_data, links_filename="links_and_skus.txt", skus_filename="skus.txt"):
    try:
        with open(links_filename, "w", encoding="utf-8") as f:
            all_skus = set()
            for category, (links, skus) in categories_data.items():
                f.write(f"\n=== Category: {category} ===\n")
                f.write("Links:\n")
                for link in links:
                    f.write(f"{link}\n")
                f.write(f"\nSKUs: {skus}\n")
                all_skus.update(skus)

        print(f"Successfully saved links and SKUs to {links_filename}")

        with open(skus_filename, "w", encoding="utf-8") as f:
            for sku in all_skus:
                f.write(f"{sku}\n")
        print(f"Successfully saved {len(all_skus)} SKUs to {skus_filename}")

    except Exception as e:
        print(f"Failed to save to files: {e}")

def main():
    print("Starting Adidas SKU scraper for men's T-shirts, polo shirts, and jackets...")

    # Set up Selenium WebDriver
    # Replace with your chromedriver path if not in PATH
    # service = Service("/path/to/chromedriver")
    service = Service()
    options = webdriver.ChromeOptions()
    options.add_argument("--headless")  # Run in headless mode
    options.add_argument("--disable-gpu")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-dev-shm-usage")
    options.add_argument("user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
    driver = webdriver.Chrome(service=service, options=options)

    try:
        categories = [
            "https://www.adidas.jp/%E3%83%A1%E3%83%B3%E3%82%BA-%E3%82%A6%E3%82%A7%E3%82%A2%E3%83%BB%E6%9C%8D-t%E3%82%B7%E3%83%A3%E3%83%84",
            "https://www.adidas.jp/%E3%83%A1%E3%83%B3%E3%82%BA-%E3%82%A6%E3%82%A7%E3%82%A2%E3%83%BB%E6%9C%8D-%E3%83%9D%E3%83%BC%E3%83%AD%E3%82%B7%E3%83%A3%E3%83%84",
            "https://www.adidas.jp/%E3%83%A1%E3%83%B3%E3%82%BA-%E3%82%A6%E3%82%A7%E3%82%A2%E3%83%BB%E6%9C%8D-%E3%82%B8%E3%83%A3%E3%83%BC%E3%82%B8",
        ]

        categories_data = {}
        for category in categories:
            print(f"Scraping category: {category}")
            links, skus = scrape_category(category, driver)
            categories_data[category] = (links, skus)
            time.sleep(random.uniform(2, 5))

        total_skus = set()
        for _, (_, skus) in categories_data.items():
            total_skus.update(skus)
        print(f"Collected {len(total_skus)} unique SKUs across all categories")

        print("Saving links and SKUs to links_and_skus.txt and skus.txt...")
        save_links_and_skus(categories_data)

    finally:
        driver.quit()

if __name__ == "__main__":
    main()
