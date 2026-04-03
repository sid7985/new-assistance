#!/usr/bin/env python3
"""
YouTube Selenium Automation Script
Searches for "maula mere maula" and plays the first video result.
"""

import sys
import time
import logging
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import (
    TimeoutException,
    NoSuchElementException,
    WebDriverException,
    ElementClickInterceptedException,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def setup_chrome_options() -> Options:
    """Configure Chrome options for optimal automation."""
    options = Options()
    options.add_argument("--start-maximized")
    options.add_argument("--disable-blink-features=AutomationControlled")
    options.add_argument("--disable-extensions")
    options.add_argument("--disable-popup-blocking")
    options.add_argument("--disable-notifications")
    options.add_argument("--disable-infobars")
    options.add_argument("--disable-dev-shm-usage")
    options.add_argument("--no-sandbox")
    options.add_argument("--disable-gpu")
    options.add_experimental_option("excludeSwitches", ["enable-automation"])
    options.add_experimental_option("useAutomationExtension", False)
    return options


def create_driver(options: Options) -> webdriver.Chrome:
    """Create and return a Chrome WebDriver instance."""
    try:
        driver = webdriver.Chrome(options=options)
        driver.execute_cdp_cmd("Page.addScriptToEvaluateOnNewDocument", {
            "source": """
                Object.defineProperty(navigator, 'webdriver', {
                    get: () => undefined
                })
            """
        })
        logger.info("Chrome WebDriver initialized successfully")
        return driver
    except WebDriverException as e:
        logger.error(f"Failed to create WebDriver: {e}")
        raise


def navigate_to_youtube(driver: webdriver.Chrome) -> None:
    """Navigate to YouTube homepage."""
    try:
        logger.info("Navigating to youtube.com")
        driver.get("https://www.youtube.com")
        WebDriverWait(driver, 15).until(
            EC.presence_of_element_located((By.NAME, "search"))
        )
        logger.info("YouTube homepage loaded")
    except TimeoutException:
        logger.error("Timeout waiting for YouTube to load")
        raise
    except WebDriverException as e:
        logger.error(f"Failed to navigate to YouTube: {e}")
        raise


def search_video(driver: webdriver.Chrome, query: str) -> None:
    """Search for a video on YouTube."""
    try:
        logger.info(f"Searching for: {query}")
        search_box = WebDriverWait(driver, 10).until(
            EC.element_to_be_clickable((By.NAME, "search"))
        )
        search_box.clear()
        search_box.send_keys(query)
        search_box.send_keys(Keys.RETURN)
        logger.info("Search submitted")
    except TimeoutException:
        logger.error("Timeout waiting for search box")
        raise
    except WebDriverException as e:
        logger.error(f"Failed to search: {e}")
        raise


def wait_for_results(driver: webdriver.Chrome, timeout: int = 15) -> None:
    """Wait for search results to load."""
    try:
        WebDriverWait(driver, timeout).until(
            EC.presence_of_element_located((By.ID, "contents"))
        )
        time.sleep(2)
        logger.info("Search results loaded")
    except TimeoutException:
        logger.error("Timeout waiting for search results")
        raise


def click_first_video(driver: webdriver.Chrome) -> None:
    """Click on the first video result."""
    try:
        first_video = WebDriverWait(driver, 10).until(
            EC.element_to_be_clickable((By.XPATH, 
                "//ytd-video-renderer[1]//a[@id='video-title'] | "
                "//a[@id='video-title'][1] | "
                "//yt-formatted-string[contains(@class, 'title')]/.."
            ))
        )
        first_video.click()
        logger.info("Clicked on first video result")
    except (TimeoutException, NoSuchElementException) as e:
        logger.error(f"Failed to find or click first video: {e}")
        raise


def handle_exception(driver: webdriver.Chrome, e: Exception) -> None:
    """Handle exceptions and provide debug information."""
    logger.error(f"Exception occurred: {type(e).__name__}: {e}")
    try:
        if driver.current_url:
            logger.error(f"Current URL: {driver.current_url}")
    except WebDriverException:
        pass


def cleanup(driver: webdriver.Chrome) -> None:
    """Clean up WebDriver resources."""
    if driver:
        try:
            driver.quit()
            logger.info("WebDriver quit successfully")
        except WebDriverException as e:
            logger.error(f"Error during cleanup: {e}")


def main():
    """Main function to run YouTube automation."""
    driver = None
    query = "maula mere maula"

    try:
        options = setup_chrome_options()
        driver = create_driver(options)
        driver.set_page_load_timeout(30)

        navigate_to_youtube(driver)
        search_video(driver, query)
        wait_for_results(driver)
        click_first_video(driver)

        logger.info("Video playback started successfully")
        logger.info("Press Ctrl+C to exit")

        while True:
            time.sleep(1)

    except KeyboardInterrupt:
        logger.info("User interrupted the script")
    except Exception as e:
        if driver:
            handle_exception(driver, e)
        logger.error(f"Script failed: {e}")
        sys.exit(1)
    finally:
        cleanup(driver)


if __name__ == "__main__":
    main()