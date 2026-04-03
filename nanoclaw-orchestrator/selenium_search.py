#!/usr/bin/env python3
"""
Selenium Search Script
Automates Google search and extracts result titles using Selenium WebDriver.
"""

import logging
import sys
from typing import Optional

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import (
    TimeoutException,
    WebDriverException,
    NoSuchElementException,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

SEARCH_QUERY = "Python Selenium automation"
NUM_RESULTS = 5
TIMEOUT = 10


def create_chrome_options(headless: bool = False) -> Options:
    """
    Create and configure Chrome options.
    
    Args:
        headless: Whether to run in headless mode.
        
    Returns:
        Configured ChromeOptions object.
    """
    options = Options()
    
    # Headless mode options (commented out for visibility)
    # if headless:
    #     options.add_argument("--headless=new")
    #     options.add_argument("--disable-gpu")
    
    # Standard options
    options.add_argument("--start-maximized")
    options.add_argument("--disable-blink-features=AutomationControlled")
    options.add_argument("--disable-extensions")
    options.add_argument("--disable-popup-blocking")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-dev-shm-usage")
    
    return options


def init_webdriver(options: Optional[Options] = None) -> webdriver.Chrome:
    """
    Initialize and return Chrome WebDriver.
    
    Args:
        options: ChromeOptions to use.
        
    Returns:
        Initialized Chrome WebDriver instance.
    """
    if options is None:
        options = create_chrome_options()
    
    driver = webdriver.Chrome(service=Service(), options=options)
    driver.implicitly_wait(TIMEOUT)
    logger.info("WebDriver initialized")
    
    return driver


def navigate_to_google(driver: webdriver.Chrome) -> None:
    """
    Navigate to Google homepage.
    
    Args:
        driver: WebDriver instance.
    """
    logger.info("Navigating to Google")
    driver.get("https://www.google.com")
    logger.info(f"Loaded: {driver.title}")


def accept_cookies(driver: webdriver.Chrome) -> None:
    """
    Accept Google cookies if cookie banner appears.
    
    Args:
        driver: WebDriver instance.
    """
    try:
        accept_button = WebDriverWait(driver, 5).until(
            EC.element_to_be_clickable((By.XPATH, "//button[contains(., 'Accept all')]"))
        )
        accept_button.click()
        logger.info("Accepted cookies")
    except Exception:
        logger.info("No cookie banner found")


def perform_search(driver: webdriver.Chrome, query: str) -> None:
    """
    Enter search query in Google's search box.
    
    Args:
        driver: WebDriver instance.
        query: Search query string.
    """
    logger.info(f"Searching for: {query}")
    
    search_box = WebDriverWait(driver, TIMEOUT).until(
        EC.presence_of_element_located((By.NAME, "q"))
    )
    
    search_box.clear()
    search_box.send_keys(query)
    search_box.send_keys(Keys.RETURN)
    
    logger.info("Search submitted")


def extract_result_titles(driver: webdriver.Chrome, num_results: int) -> list[str]:
    """
    Extract titles of search results.
    
    Args:
        driver: WebDriver instance.
        num_results: Number of results to extract.
        
    Returns:
        List of result titles.
    """
    logger.info(f"Extracting first {num_results} result titles")
    
    WebDriverWait(driver, TIMEOUT).until(
        EC.presence_of_element_located((By.ID, "search"))
    )
    
    result_titles = []
    
    for i in range(1, num_results + 1):
        try:
            title_element = driver.find_element(
                By.XPATH,
                f"(//div[@class='g']//h3)[{i}]"
            )
            title = title_element.text
            result_titles.append(title)
            logger.info(f"Result {i}: {title}")
        except NoSuchElementException:
            logger.warning(f"Could not find result {i}")
            continue
    
    return result_titles


def print_results(titles: list[str]) -> None:
    """
    Print the extracted result titles.
    
    Args:
        titles: List of result titles.
    """
    print("\n" + "=" * 60)
    print("SEARCH RESULTS")
    print("=" * 60)
    
    for idx, title in enumerate(titles, start=1):
        print(f"{idx}. {title}")
    
    print("=" * 60 + "\n")


def close_browser(driver: Optional[webdriver.Chrome]) -> None:
    """
    Close the browser gracefully.
    
    Args:
        driver: WebDriver instance to close.
    """
    if driver:
        try:
            driver.quit()
            logger.info("Browser closed successfully")
        except Exception as e:
            logger.error(f"Error closing browser: {e}")


def main() -> None:
    """
    Main execution function.
    """
    driver = None
    
    try:
        logger.info("Starting Selenium search script")
        
        options = create_chrome_options(headless=False)
        driver = init_webdriver(options)
        
        navigate_to_google(driver)
        accept_cookies(driver)
        perform_search(driver, SEARCH_QUERY)
        
        titles = extract_result_titles(driver, NUM_RESULTS)
        print_results(titles)
        
        logger.info("Script completed successfully")
        
    except WebDriverException as e:
        logger.error(f"WebDriver error: {e}")
        sys.exit(1)
        
    except TimeoutException as e:
        logger.error(f"Timeout error: {e}")
        sys.exit(1)
        
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        sys.exit(1)
        
    finally:
        close_browser(driver)


if __name__ == "__main__":
    main()