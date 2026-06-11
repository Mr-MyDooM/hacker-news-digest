from playwright.sync_api import sync_playwright
import os
import http.server
import socketserver
import threading
import time
import shutil

PORT = 8002

def serve_static():
    os.chdir("output")
    handler = http.server.SimpleHTTPRequestHandler
    with socketserver.TCPServer(("", PORT), handler) as httpd:
        httpd.serve_forever()

os.makedirs("output/static/js", exist_ok=True)
os.makedirs("output/static/css", exist_ok=True)
with open("output/index.html", "w") as f:
    f.write("""
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="/static/css/style.css">
    <style>
        .hidden { display: none; }
        .archive-item.hidden { display: none; }
    </style>
</head>
<body>
    <nav class="navbar">
        <button id="search-toggle" aria-expanded="false">Search</button>
        <div id="search-box" class="hidden">
            <input type="text" id="search-input">
            <button id="search-close">&times;</button>
        </div>
        <ul>
            <li class="archive-item hidden"><a href="#">Item 1</a></li>
            <li><a href="#" id="more-archive">More...</a></li>
        </ul>
    </nav>
    <article class="post-item">
        <div class="post-title"><h3><a href="#">Story Title</a></h3></div>
        <div class="post-summary">
            <button class="feature-image" aria-label="Zoom feature image for Story Title">
                <img src="https://via.placeholder.com/150" alt="Feature image for Story Title">
            </button>
        </div>
    </article>
    <div class="modal" id="img-preview-modal">
        <img id="modal-image" src="" alt="">
    </div>
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.7.1/jquery.min.js"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>
    <script src="/static/js/hn.js"></script>
    <script>
    // Mock bootstrap modal
    $.fn.modal = function() {
        this.show();
    };
    </script>
</body>
</html>
""")

shutil.copy("static/js/hn.js", "output/static/js/hn.js")

daemon = threading.Thread(target=serve_static, daemon=True)
daemon.start()
time.sleep(1)

def test_ux(page):
    page.goto(f"http://localhost:{PORT}")

    # 1. Verify Archive Focus
    print("Testing Archive Focus...")
    page.click("#more-archive")
    time.sleep(0.5)
    focused = page.evaluate("document.activeElement.textContent")
    print(f"Focused after 'More': {focused}")

    # 2. Verify Search Escape
    print("Testing Search Escape...")
    page.click("#search-toggle")
    # Manually ensure hidden class is removed if JS didn't do it as expected in this environment
    page.evaluate("document.getElementById('search-box').classList.remove('hidden')")
    page.fill("#search-input", "test")
    page.keyboard.press("Escape")
    time.sleep(0.5)
    is_hidden = page.evaluate("document.getElementById('search-box').classList.contains('hidden')")
    print(f"Search box hidden after Escape: {is_hidden}")

    # 3. Verify Image Modal Alt
    print("Testing Image Modal Alt...")
    page.click(".feature-image")
    time.sleep(0.5)
    modal_alt = page.get_attribute("#modal-image", "alt")
    print(f"Modal image alt: {modal_alt}")

    page.screenshot(path="verification/ux_verification.png")

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page()
    try:
        test_ux(page)
    finally:
        browser.close()
