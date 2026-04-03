#!/usr/bin/env python3
"""
Selenium Web Automation Script
Automates browser tasks using Selenium WebDriver with Chrome.
"""

import logging
import os
import sys
import time
from typing import Optional

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import (
    TimeoutException,
    WebDriverException,
    NoSuchElementException,
)

# =============================================================================
# Configuration Constants
# =============================================================================

TARGET_URL: str = "https://www.example.com"
TIMEOUT_SECONDS: int = 10
WINDOW_WIDTH: int = 1920
WINDOW_HEIGHT: int = 1080
HEADLESS_MODE: bool = False  # Set to True for headless execution
SCREENSHOT_DIR: str = "screenshots"
LOG_LEVEL: int = logging.INFO

# Chrome options configuration
CHROME_OPTIONS_ARGS: list[str] = [
    "--start-maximized",
    "--disable-blink-features=AutomationControlled",
    "--disable-extensions",
    "--disable-popup-blocking",
    "--no-sandbox",
    "--disable-dev-shm-usage",
]


# =============================================================================
# Logging Setup
# =============================================================================

def setup_logging() -> logging.Logger:
    """Configure and return a logger instance."""
    logger = logging.getLogger("selenium_automation")
    logger.setLevel(LOG_LEVEL)

    if not logger.handlers:
        handler = logging.StreamHandler(sys.stdout)
        handler.setLevel(LOG_LEVEL)
        formatter = logging.Formatter(
            "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
        )
        handler.setFormatter(formatter)
        logger.addHandler(handler)

    return logger


logger = setup_logging()


# =============================================================================
# WebDriver Setup
# =============================================================================

def create_chrome_options(headless: bool = HEADLESS_MODE) -> Options:
    """
    Create and configure Chrome options.

    Args:
        headless: Whether to run in headless mode.

    Returns:
        Configured ChromeOptions object.
    """
    options = Options()

    if headless:
        options.add_argument("--headless=new")
        logger.info("Running in headless mode")

    for arg in CHROME_OPTIONS_ARGS:
        options.add_argument(arg)

    options.add_argument(f"--window-size={WINDOW_WIDTH},{WINDOW_HEIGHT}")
    options.add_experimental_option("excludeSwitches", ["enable-automation"])
    options.add_experimental_option("useAutomationExtension", False)

    return options


def init_webdriver(
    options: Optional[Options] = None,
    driver_path: Optional[str] = None,
) -> webdriver.Chrome:
    """
    Initialize and return Chrome WebDriver.

    Args:
        options: ChromeOptions to use. If None, creates default options.
        driver_path: Optional path to chromedriver executable.

    Returns:
        Initialized Chrome WebDriver instance.
    """
    if options is None:
        options = create_chrome_options()

    service = Service(executable_path=driver_path) if driver_path else Service()

    logger.info("Initializing Chrome WebDriver")
    driver = webdriver.Chrome(service=service, options=options)

    driver.implicitly_wait(TIMEOUT_SECONDS)
    logger.info(f"WebDriver initialized with implicit wait of {TIMEOUT_SECONDS}s")

    return driver


# =============================================================================
# Page Interaction Functions
# =============================================================================

def navigate_to_url(driver: webdriver.Chrome, url: str) -> None:
    """
    Navigate to the specified URL.

    Args:
        driver: WebDriver instance.
        url: Target URL to navigate to.
    """
    logger.info(f"Navigating to: {url}")
    driver.get(url)
    logger.info(f"Successfully navigated to: {driver.current_url}")


def wait_for_page_load(driver: webdriver.Chrome, timeout: int = TIMEOUT_SECONDS) -> bool:
    """
    Wait for page to fully load.

    Args:
        driver: WebDriver instance.
        timeout: Maximum wait time in seconds.

    Returns:
        True if page loaded successfully, False otherwise.
    """
    try:
        wait = WebDriverWait(driver, timeout)
        wait.until(lambda d: d.execute_script("return document.readyState") == "complete")
        logger.info("Page fully loaded")
        return True
    except TimeoutException:
        logger.warning("Page load timeout - continuing anyway")
        return False


def get_page_title(driver: webdriver.Chrome) -> str:
    """
    Extract the page title.

    Args:
        driver: WebDriver instance.

    Returns:
        The page title text.
    """
    title = driver.title
    logger.info(f"Page title: {title}")
    return title


def extract_all_links(driver: webdriver.Chrome) -> list[dict]:
    """
    Find all links on the page and extract their href attributes.

    Args:
        driver: WebDriver instance.

    Returns:
        List of dictionaries containing link information.
    """
    logger.info("Extracting all links from the page")

    links = driver.find_elements(By.TAG_NAME, "a")
    link_data = []

    for idx, link in enumerate(links, start=1):
        try:
            href = link.get_attribute("href")
            text = link.text.strip()
            link_data.append({
                "index": idx,
                "href": href,
                "text": text if text else "[no text]",
            })
            if href:
                logger.debug(f"Link {idx}: {href}")
        except Exception as e:
            logger.warning(f"Error extracting link {idx}: {e}")
            continue

    logger.info(f"Found {len(link_data)} links on the page")
    return link_data


def print_links(link_data: list[dict]) -> None:
    """
    Print all extracted links in a formatted manner.

    Args:
        link_data: List of link dictionaries.
    """
    print("\n" + "=" * 60)
    print("EXTRACTED LINKS")
    print("=" * 60)

    for link in link_data:
        print(f"\nLink #{link['index']}")
        print(f"  Text: {link['text']}")
        print(f"  HREF: {link['href']}")

    print("\n" + "=" * 60 + "\n")


# =============================================================================
# Screenshot Functions
# =============================================================================

def ensure_screenshot_dir() -> None:
    """Create screenshots directory if it doesn't exist."""
    if not os.path.exists(SCREENSHOT_DIR):
        os.makedirs(SCREENSHOT_DIR)
        logger.info(f"Created screenshot directory: {SCREENSHOT_DIR}")


def take_screenshot(driver: webdriver.Chrome, filename: str) -> str:
    """
    Take a screenshot of the current page.

    Args:
        driver: WebDriver instance.
        filename: Name for the screenshot file.

    Returns:
        Full path to the saved screenshot.
    """
    ensure_screenshot_dir()

    filepath = os.path.join(SCREENSHOT_DIR, filename)
    driver.save_screenshot(filepath)
    logger.info(f"Screenshot saved to: {filepath}")

    return filepath


# =============================================================================
# Browser Cleanup
# =============================================================================

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


# =============================================================================
# Main Orchestration
# =============================================================================

def main() -> None:
    """
    Main function that orchestrates the Selenium automation flow.
    """
    logger.info("=" * 60)
    logger.info("Starting Selenium Automation Script")
    logger.info("=" * 60)

    driver: Optional[webdriver.Chrome] = None

    try:
        # Step 1: Setup Chrome with options
        logger.info("\n[Step 1] Setting up Chrome browser with options")
        options = create_chrome_options(headless=HEADLESS_MODE)
        driver = init_webdriver(options)

        # Step 2: Navigate to target website
        logger.info("\n[Step 2] Navigating to target website")
        navigate_to_url(driver, TARGET_URL)
        wait_for_page_load(driver)

        # Step 3: Extract page title
        logger.info("\n[Step 3] Extracting page title")
        title = get_page_title(driver)
        print(f"\n>>> Page Title: {title}\n")

        # Step 4: Find and print all links
        logger.info("\n[Step 4] Finding and extracting all links")
        link_data = extract_all_links(driver)
        print_links(link_data)

        # Step 5: Take a screenshot
        logger.info("\n[Step 5] Taking screenshot of the page")
        screenshot_filename = f"example_page_{int(time.time())}.png"
        take_screenshot(driver, screenshot_filename)

        logger.info("\n" + "=" * 60)
        logger.info("Automation completed successfully!")
        logger.info("=" * 60)

    except WebDriverException as e:
        logger.error(f"WebDriver error: {e}")
        raise

    except TimeoutException as e:
        logger.error(f"Timeout error: {e}")
        raise

    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        raise

    finally:
        # Step 6: Close browser gracefully
        logger.info("\n[Step 6] Closing browser")
        close_browser(driver)


if __name__ == "__main__":
    main()