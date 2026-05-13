import asyncio
from playwright.async_api import async_playwright
import os
import subprocess
import time

async def verify():
    # Start server
    server_process = subprocess.Popen(["python3", "-m", "http.server", "8000", "--directory", "output"])
    time.sleep(2) # Wait for server

    try:
        async with async_playwright() as p:
            browser = await p.chromium.launch()
            page = await browser.new_page()
            await page.goto("http://localhost:8000")

            # 1. Trigger search and type non-existent query
            await page.keyboard.press("/")
            await page.fill("#search-input", "xyzqweasdzxc")
            await page.wait_for_timeout(500) # wait for doSearch debounce

            # Capture no results state
            os.makedirs("verification", exist_ok=True)
            await page.screenshot(path="verification/search_no_results_final.png")
            print("Captured search_no_results_final.png")

            # 2. Click Clear Search
            await page.click("#search-clear")
            await page.wait_for_timeout(500)
            await page.screenshot(path="verification/search_cleared_final.png")
            print("Captured search_cleared_final.png")

            # 3. Test Escape key logic (Regression check)
            # Type something and press escape
            await page.fill("#search-input", "test")
            await page.focus("#search-input")
            await page.keyboard.press("Escape")
            # Should clear input
            val = await page.input_value("#search-input")
            print(f"Value after first escape: '{val}'")
            if val != "":
                print("FAILED: Escape did not clear search input")

            await page.keyboard.press("Escape")
            # Should hide box
            is_hidden = await page.eval_on_selector("#search-box", "el => el.classList.contains('hidden')")
            print(f"Search box hidden after second escape: {is_hidden}")

            await browser.close()
    finally:
        server_process.terminate()

if __name__ == "__main__":
    asyncio.run(verify())
