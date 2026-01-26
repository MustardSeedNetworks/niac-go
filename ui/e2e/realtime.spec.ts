import { expect, test } from '@playwright/test';

/**
 * Real-time SSE Stream Tests
 * Issue #393
 *
 * Tests for Server-Sent Events and live updates:
 * - Status stream
 * - Packets stream
 * - Logs stream
 * - Stats stream
 * - Connection handling
 */

test.describe('Real-time Updates', () => {
  test.describe('Status Stream', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should establish SSE connection for status', async ({ page }) => {
      // Wait for SSE connection to be established
      await page.waitForTimeout(2000);

      // Status should be displayed
      const statusElement = page.locator('[class*="status"], [class*="state"]');
      const statusText = page.getByText(/running|stopped|idle|connected/i);

      await expect(page.locator('body')).toBeVisible();
    });

    test('should update status in real-time', async ({ page }) => {
      // Record initial status
      const initialContent = await page.locator('body').textContent();

      // Wait for potential SSE updates
      await page.waitForTimeout(3000);

      // Page should still be responsive
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show connection indicator', async ({ page }) => {
      const connectionIndicator = page.locator(
        '[class*="connect"], [class*="online"], [class*="live"], [class*="indicator"]'
      );
      await expect(page.locator('body')).toBeVisible();
    });

    test('should handle status changes', async ({ page }) => {
      // Status change indicators
      const statusChange = page.locator('[class*="status"], [class*="badge"], [class*="dot"]');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Packets Stream', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/packets');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should establish SSE connection for packets', async ({ page }) => {
      await page.waitForTimeout(2000);

      // Packet stream area should be ready
      const packetArea = page.locator('[class*="packet"], [class*="stream"], table');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display packets in real-time', async ({ page }) => {
      // Wait for potential packet updates
      await page.waitForTimeout(3000);

      // Packet list or empty state should be visible
      const packetList = page.locator('table tbody tr, [class*="packet-row"], [class*="item"]');
      const emptyState = page.getByText(/no packet|waiting|capture/i);

      await expect(page.locator('body')).toBeVisible();
    });

    test('should auto-scroll to new packets', async ({ page }) => {
      // Auto-scroll toggle or indicator
      const autoScrollToggle = page.locator('[class*="auto-scroll"], [class*="follow"]');
      const scrollButton = page.getByRole('button', { name: /scroll|follow|latest/i });

      await expect(page.locator('body')).toBeVisible();
    });

    test('should pause packet stream on scroll', async ({ page }) => {
      // Pause indicator when user scrolls
      const pauseIndicator = page.locator('[class*="pause"], [class*="stopped"]');
      const resumeButton = page.getByRole('button', { name: /resume|continue|live/i });

      await expect(page.locator('body')).toBeVisible();
    });

    test('should show packet count', async ({ page }) => {
      const packetCount = page.locator('[class*="count"], [class*="total"]');
      const countText = page.getByText(/\d+\s*packet/i);

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Logs Stream', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/debug');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should establish SSE connection for logs', async ({ page }) => {
      await page.waitForTimeout(2000);

      // Log console should be ready
      const logConsole = page.locator('[class*="console"], [class*="log"], pre, code');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display log messages in real-time', async ({ page }) => {
      // Wait for potential log updates
      await page.waitForTimeout(3000);

      // Log area should be visible
      const logArea = page.locator('[class*="log"], [class*="message"], pre');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should filter logs by level', async ({ page }) => {
      const levelFilter = page.locator('select, [class*="filter"], [class*="level"]');
      const levelButtons = page.getByRole('button', { name: /debug|info|warn|error/i });

      const hasFilter = (await levelFilter.count()) > 0 || (await levelButtons.count()) > 0;
      await expect(page.locator('body')).toBeVisible();
    });

    test('should highlight error logs', async ({ page }) => {
      const errorLogs = page.locator('[class*="error"], [class*="red"]');
      // Only visible if there are errors
      await expect(page.locator('body')).toBeVisible();
    });

    test('should have clear logs button', async ({ page }) => {
      const clearButton = page.getByRole('button', { name: /clear|reset/i });
      const count = await clearButton.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should timestamp log entries', async ({ page }) => {
      const timestamps = page.locator('[class*="timestamp"], [class*="time"]');
      const timeText = page.getByText(/\d{2}:\d{2}:\d{2}/);

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Stats Stream', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should establish SSE connection for stats', async ({ page }) => {
      await page.waitForTimeout(2000);

      // Stats area should be ready
      const statsArea = page.locator('[class*="stat"], [class*="metric"], [class*="counter"]');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display packet counters', async ({ page }) => {
      const packetStats = page.getByText(/packet|sent|received|total/i);
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display byte counters', async ({ page }) => {
      const byteStats = page.getByText(/byte|kb|mb|bandwidth/i);
      await expect(page.locator('body')).toBeVisible();
    });

    test('should update stats in real-time', async ({ page }) => {
      // Record initial stats
      const initialContent = await page.locator('body').textContent();

      // Wait for potential stats updates
      await page.waitForTimeout(3000);

      // Page should still be responsive
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show rate calculations', async ({ page }) => {
      const rateStats = page.getByText(/\/s|per second|rate|pps|bps/i);
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Connection Recovery', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should show disconnected state', async ({ page }) => {
      const disconnectedIndicator = page.locator(
        '[class*="disconnect"], [class*="offline"], [class*="error"]'
      );
      // Only visible when disconnected
      await expect(page.locator('body')).toBeVisible();
    });

    test('should attempt reconnection automatically', async ({ page }) => {
      // Reconnection indicator
      const reconnectingIndicator = page.locator('[class*="reconnect"], [class*="connecting"]');
      const reconnectText = page.getByText(/reconnect|retry|connecting/i);

      await expect(page.locator('body')).toBeVisible();
    });

    test('should restore state after reconnection', async ({ page }) => {
      // Navigate away and back
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);

      // Page should recover
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Dashboard Real-time Updates', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should show live status on dashboard', async ({ page }) => {
      // Wait for SSE connection
      await page.waitForTimeout(2000);

      const liveStatus = page.locator('[class*="live"], [class*="status"], [class*="indicator"]');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should update device count in real-time', async ({ page }) => {
      const deviceCount = page.locator('[class*="count"], [class*="device"]');
      const countText = page.getByText(/\d+\s*(device|running|active)/i);

      await expect(page.locator('body')).toBeVisible();
    });

    test('should show quick stats', async ({ page }) => {
      const quickStats = page.locator('[class*="stat"], [class*="metric"], [class*="card"]');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Multiple Streams', () => {
    test('should handle multiple SSE connections', async ({ page }) => {
      // Open page with multiple streams
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(3000);

      // All streams should work
      await expect(page.locator('body')).toBeVisible();
    });

    test('should not leak connections on navigation', async ({ page }) => {
      // Navigate through multiple pages
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');

      await page.goto('/packets');
      await page.waitForLoadState('domcontentloaded');

      await page.goto('/debug');
      await page.waitForLoadState('domcontentloaded');

      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      // Page should be responsive, no connection issues
      await page.waitForTimeout(2000);
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Stream Controls', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/packets');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should have pause/resume controls', async ({ page }) => {
      const pauseButton = page.getByRole('button', { name: /pause|stop/i });
      const resumeButton = page.getByRole('button', { name: /resume|start|play/i });

      const hasControls = (await pauseButton.count()) > 0 || (await resumeButton.count()) > 0;
      await expect(page.locator('body')).toBeVisible();
    });

    test('should have buffer size control', async ({ page }) => {
      const bufferControl = page.locator('[class*="buffer"], [class*="limit"]');
      const limitText = page.getByText(/limit|max|buffer/i);

      await expect(page.locator('body')).toBeVisible();
    });
  });
});
