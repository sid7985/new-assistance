"""
Selenium WebDriver Setup Module

This module provides a WebDriverManager class for initializing Selenium WebDriver
instances with proper error handling and automatic driver version management.

Installation:
    pip install selenium
    pip install webdriver-manager
"""

import logging
from typing import Optional

from selenium import webdriver
from selenium.common.exceptions import WebDriverException
from webdriver_manager.chrome import ChromeDriverManager
from webdriver_manager.firefox import GeckoDriverManager
from webdriver_manager.core.os_manager import OperationSystemManager

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class WebDriverManager:
    """
    A class to manage Selenium WebDriver initialization with support for
    Chrome and Firefox browsers. Uses webdriver-manager for automatic
    driver version management.
    """

    @staticmethod
    def initialize_chrome_driver() -> webdriver.Chrome:
        """
        Initialize Chrome WebDriver with automatic driver management.

        Returns:
            webdriver.Chrome: Configured Chrome WebDriver instance.

        Raises:
            WebDriverException: If Chrome or ChromeDriver is not found.
            FileNotFoundException: If ChromeDriver executable is missing.
        """
        try:
            logger.info("Initializing Chrome WebDriver with webdriver-manager...")

            service = webdriver.ChromeService(ChromeDriverManager().install())
            options = webdriver.ChromeOptions()
            options.add_argument("--start-maximized")
            options.add_argument("--disable-extensions")
            options.add_argument("--no-sandbox")
            options.add_argument("--disable-dev-shm-usage")

            driver = webdriver.Chrome(service=service, options=options)
            logger.info("Chrome WebDriver initialized successfully")
            return driver

        except WebDriverException as e:
            error_msg = (
                "Chrome or ChromeDriver not found. Please ensure Google Chrome is installed.\n"
                "Installation instructions:\n"
                "1. Download ChromeDriver from: https://chromedriver.chromium.org/\n"
                "2. Add ChromeDriver to your system PATH\n"
                "3. Or run: pip install webdriver-manager (automatic version management)"
            )
            logger.error(f"Chrome WebDriver initialization failed: {e}")
            raise WebDriverException(error_msg) from e

        except FileNotFoundException as e:
            error_msg = (
                "ChromeDriver executable not found.\n"
                "Download from: https://chromedriver.chromium.org/\n"
                "Add the executable to your system PATH or ensure it's in the same directory."
            )
            logger.error(f"ChromeDriver file not found: {e}")
            raise FileNotFoundException(error_msg) from e

    @staticmethod
    def initialize_firefox_driver() -> webdriver.Firefox:
        """
        Initialize Firefox WebDriver with automatic driver management.

        Returns:
            webdriver.Firefox: Configured Firefox WebDriver instance.

        Raises:
            WebDriverException: If Firefox or GeckoDriver is not found.
            FileNotFoundException: If GeckoDriver executable is missing.
        """
        try:
            logger.info("Initializing Firefox WebDriver with webdriver-manager...")

            service = webdriver.FirefoxService(GeckoDriverManager().install())
            options = webdriver.FirefoxOptions()
            options.add_argument("--start-maximized")
            options.add_argument("--disable-extensions")

            driver = webdriver.Firefox(service=service, options=options)
            logger.info("Firefox WebDriver initialized successfully")
            return driver

        except WebDriverException as e:
            error_msg = (
                "Firefox or GeckoDriver not found. Please ensure Mozilla Firefox is installed.\n"
                "Installation instructions:\n"
                "1. Download GeckoDriver from: https://github.com/mozilla/geckodriver/releases\n"
                "2. Add GeckoDriver to your system PATH\n"
                "3. Or run: pip install webdriver-manager (automatic version management)"
            )
            logger.error(f"Firefox WebDriver initialization failed: {e}")
            raise WebDriverException(error_msg) from e

        except FileNotFoundException as e:
            error_msg = (
                "GeckoDriver executable not found.\n"
                "Download from: https://github.com/mozilla/geckodriver/releases\n"
                "Add the executable to your system PATH or ensure it's in the same directory."
            )
            logger.error(f"GeckoDriver file not found: {e}")
            raise FileNotFoundException(error_msg) from e

    @classmethod
    def initialize_driver(cls, browser: str = 'chrome') -> Optional[webdriver.Remote]:
        """
        Factory method to initialize the appropriate WebDriver based on browser type.

        Args:
            browser (str): Browser name ('chrome' or 'firefox'). Defaults to 'chrome'.

        Returns:
            Optional[webdriver.Remote]: Configured WebDriver instance.

        Raises:
            ValueError: If an unsupported browser is specified.
        """
        browser = browser.lower()

        if browser == 'chrome':
            return cls.initialize_chrome_driver()
        elif browser == 'firefox':
            return cls.initialize_firefox_driver()
        else:
            error_msg = f"Unsupported browser: '{browser}'. Supported browsers: 'chrome', 'firefox'"
            logger.error(error_msg)
            raise ValueError(error_msg)


if __name__ == "__main__":
    print("=" * 60)
    print("Selenium WebDriver Setup Demo")
    print("=" * 60)

    driver = None
    try:
        print("\nAttempting to initialize Chrome WebDriver...")
        driver = WebDriverManager.initialize_driver(browser='chrome')
        print("Chrome WebDriver initialized successfully!")

        if driver:
            driver.get("https://www.google.com")
            print(f"Page title: {driver.title}")

    except WebDriverException as e:
        print(f"\nWebDriverException: {e}")
    except FileNotFoundException as e:
        print(f"\nFileNotFoundException: {e}")
    except Exception as e:
        print(f"\nUnexpected error: {type(e).__name__}: {e}")
    finally:
        if driver:
            driver.quit()
            print("\nWebDriver closed successfully")

    print("\n" + "=" * 60)
    print("Attempting Firefox WebDriver...")
    print("=" * 60)

    driver = None
    try:
        print("\nAttempting to initialize Firefox WebDriver...")
        driver = WebDriverManager.initialize_driver(browser='firefox')
        print("Firefox WebDriver initialized successfully!")

        if driver:
            driver.get("https://www.google.com")
            print(f"Page title: {driver.title}")

    except WebDriverException as e:
        print(f"\nWebDriverException: {e}")
    except FileNotFoundException as e:
        print(f"\nFileNotFoundException: {e}")
    except Exception as e:
        print(f"\nUnexpected error: {type(e).__name__}: {e}")
    finally:
        if driver:
            driver.quit()
            print("\nWebDriver closed successfully")
