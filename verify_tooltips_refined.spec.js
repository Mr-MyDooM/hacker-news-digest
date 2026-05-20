const { test, expect } = require('@playwright/test');

test('verify tooltips refined', async ({ page }) => {
  await page.goto('http://localhost:8080');

  // Theme toggle tooltip
  await page.hover('#theme-toggle');
  await page.waitForSelector('.tooltip.in', { state: 'visible' });
  await page.screenshot({ path: 'tooltip_theme_refined.png' });

  // Search toggle tooltip
  await page.hover('#search-toggle');
  await page.waitForSelector('.tooltip.in', { state: 'visible' });
  await page.screenshot({ path: 'tooltip_search_refined.png' });

  // Share icon tooltip
  const firstShare = page.locator('.share-icon').first();
  await firstShare.hover();
  await page.waitForSelector('.tooltip.in', { state: 'visible' });
  await page.screenshot({ path: 'tooltip_share_refined.png' });

  // Test "Copied" state
  await firstShare.click();
  // The tooltip title should have changed to "Copied!"
  // Bootstrap usually recreates or updates the tooltip
  await page.waitForTimeout(200);
  await page.screenshot({ path: 'tooltip_copied_refined.png' });
});
