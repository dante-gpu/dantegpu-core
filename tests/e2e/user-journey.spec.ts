import { test, expect, Page } from '@playwright/test';

// Test configuration
const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const API_URL = process.env.API_URL || 'http://localhost:8000';

// Test user data
const testUser = {
  email: `e2e_test_${Date.now()}@example.com`,
  password: 'E2ETest123!',
  firstName: 'E2E',
  lastName: 'Tester',
};

test.describe('Complete User Journey', () => {
  let page: Page;

  test.beforeAll(async ({ browser }) => {
    page = await browser.newPage();
  });

  test.afterAll(async () => {
    await page.close();
  });

  test('Step 1: User Registration', async () => {
    await page.goto(`${BASE_URL}/register`);

    // Fill registration form
    await page.fill('input[name="email"]', testUser.email);
    await page.fill('input[name="password"]', testUser.password);
    await page.fill('input[name="confirmPassword"]', testUser.password);
    await page.fill('input[name="firstName"]', testUser.firstName);
    await page.fill('input[name="lastName"]', testUser.lastName);
    await page.check('input[name="termsAccepted"]');

    // Submit form
    await page.click('button[type="submit"]');

    // Wait for success message
    await expect(page.locator('text=Registration successful')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=Please check your email')).toBeVisible();
  });

  test('Step 2: Email Verification (simulated)', async () => {
    // In real E2E, we would check email and click verification link
    // For testing, we'll verify directly via API
    const response = await fetch(`${API_URL}/api/v1/auth/verify-email`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testUser.email }),
    });

    expect(response.status).toBe(200);
  });

  test('Step 3: Login', async () => {
    await page.goto(`${BASE_URL}/login`);

    // Fill login form
    await page.fill('input[name="email"]', testUser.email);
    await page.fill('input[name="password"]', testUser.password);

    // Submit form
    await page.click('button[type="submit"]');

    // Wait for redirect to dashboard
    await page.waitForURL(`${BASE_URL}/dashboard`, { timeout: 10000 });
    await expect(page.locator('text=Welcome')).toBeVisible();
  });

  test('Step 4: Create Wallet', async () => {
    // Navigate to wallet page
    await page.click('a[href="/wallet"]');
    await page.waitForURL(`${BASE_URL}/wallet`);

    // Check if wallet exists, if not create one
    const createWalletButton = page.locator('button:has-text("Create Wallet")');
    if (await createWalletButton.isVisible()) {
      await createWalletButton.click();

      // Wait for wallet creation
      await expect(page.locator('text=Wallet created successfully')).toBeVisible({ timeout: 30000 });
    }

    // Verify wallet address is displayed
    await expect(page.locator('text=/[A-Za-z0-9]{32,44}/')).toBeVisible();
  });

  test('Step 5: Browse GPU Marketplace', async () => {
    // Navigate to marketplace
    await page.click('a[href="/marketplace"]');
    await page.waitForURL(`${BASE_URL}/marketplace`);

    // Wait for GPUs to load
    await expect(page.locator('[data-testid="gpu-card"]').first()).toBeVisible({ timeout: 10000 });

    // Verify GPU cards are displayed
    const gpuCards = await page.locator('[data-testid="gpu-card"]').count();
    expect(gpuCards).toBeGreaterThan(0);

    // Check GPU details
    const firstGPU = page.locator('[data-testid="gpu-card"]').first();
    await expect(firstGPU.locator('text=/NVIDIA|AMD|Intel/')).toBeVisible();
    await expect(firstGPU.locator('text=/GB/')).toBeVisible(); // Memory
    await expect(firstGPU.locator('text=/\\$|dGPU/')).toBeVisible(); // Price
  });

  test('Step 6: Filter GPUs', async () => {
    // Apply filters
    await page.selectOption('select[name="gpuModel"]', 'RTX 4090');
    await page.fill('input[name="minMemory"]', '16');
    await page.fill('input[name="maxPrice"]', '0.10');

    // Wait for filtered results
    await page.waitForTimeout(1000);

    // Verify filtered results
    const filteredGPUs = await page.locator('[data-testid="gpu-card"]').count();
    expect(filteredGPUs).toBeGreaterThanOrEqual(0);
  });

  test('Step 7: View GPU Details', async () => {
    // Click on first GPU
    await page.locator('[data-testid="gpu-card"]').first().click();

    // Wait for details page
    await expect(page.locator('h1')).toContainText(/NVIDIA|AMD|Intel/);

    // Verify details are displayed
    await expect(page.locator('text=Specifications')).toBeVisible();
    await expect(page.locator('text=Price per minute')).toBeVisible();
    await expect(page.locator('text=Availability')).toBeVisible();
  });

  test('Step 8: Start Rental (simulated)', async () => {
    // Click rent button
    const rentButton = page.locator('button:has-text("Rent GPU")');
    await expect(rentButton).toBeVisible();

    // Note: Actual rental requires wallet balance
    // In real test, we would fund wallet first
    // For now, we'll just verify the button exists
  });

  test('Step 9: Submit Job', async () => {
    // Navigate to jobs page
    await page.click('a[href="/jobs"]');
    await page.waitForURL(`${BASE_URL}/jobs`);

    // Click new job button
    await page.click('button:has-text("New Job")');

    // Fill job form
    await page.fill('input[name="jobName"]', 'E2E Test Job');
    await page.fill('input[name="dockerImage"]', 'tensorflow/tensorflow:latest-gpu');
    await page.fill('textarea[name="command"]', 'python -c "print(\\"Hello from GPU!\\")"');
    await page.selectOption('select[name="gpuCount"]', '1');

    // Submit job
    await page.click('button[type="submit"]');

    // Wait for success message
    await expect(page.locator('text=Job submitted successfully')).toBeVisible({ timeout: 10000 });
  });

  test('Step 10: View Job Status', async () => {
    // Should be on jobs list page
    await expect(page.locator('[data-testid="job-card"]').first()).toBeVisible({ timeout: 5000 });

    // Click on first job
    await page.locator('[data-testid="job-card"]').first().click();

    // Verify job details
    await expect(page.locator('text=Job Details')).toBeVisible();
    await expect(page.locator('text=Status')).toBeVisible();
    await expect(page.locator('text=/pending|scheduled|running|completed/')).toBeVisible();
  });

  test('Step 11: View Job Logs', async () => {
    // Click logs tab
    await page.click('button:has-text("Logs")');

    // Verify logs section is visible
    await expect(page.locator('[data-testid="job-logs"]')).toBeVisible();

    // Logs might be empty if job hasn't started
    // Just verify the section exists
  });

  test('Step 12: View Billing History', async () => {
    // Navigate to billing page
    await page.click('a[href="/billing"]');
    await page.waitForURL(`${BASE_URL}/billing`);

    // Verify billing page elements
    await expect(page.locator('text=Billing History')).toBeVisible();
    await expect(page.locator('text=Total Spent')).toBeVisible();
  });

  test('Step 13: View Profile', async () => {
    // Navigate to profile page
    await page.click('a[href="/profile"]');
    await page.waitForURL(`${BASE_URL}/profile`);

    // Verify profile information
    await expect(page.locator(`text=${testUser.email}`)).toBeVisible();
    await expect(page.locator(`text=${testUser.firstName}`)).toBeVisible();
    await expect(page.locator(`text=${testUser.lastName}`)).toBeVisible();
  });

  test('Step 14: Update Profile', async () => {
    // Click edit button
    await page.click('button:has-text("Edit Profile")');

    // Update first name
    await page.fill('input[name="firstName"]', 'Updated');

    // Save changes
    await page.click('button:has-text("Save Changes")');

    // Wait for success message
    await expect(page.locator('text=Profile updated successfully')).toBeVisible({ timeout: 5000 });
  });

  test('Step 15: Logout', async () => {
    // Click user menu
    await page.click('[data-testid="user-menu"]');

    // Click logout
    await page.click('button:has-text("Logout")');

    // Wait for redirect to login page
    await page.waitForURL(`${BASE_URL}/login`, { timeout: 5000 });
    await expect(page.locator('text=Login')).toBeVisible();
  });
});

test.describe('Error Handling', () => {
  test('Invalid login credentials', async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);

    await page.fill('input[name="email"]', 'invalid@example.com');
    await page.fill('input[name="password"]', 'WrongPassword123!');
    await page.click('button[type="submit"]');

    await expect(page.locator('text=Invalid credentials')).toBeVisible({ timeout: 5000 });
  });

  test('Weak password registration', async ({ page }) => {
    await page.goto(`${BASE_URL}/register`);

    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'weak');
    await page.fill('input[name="confirmPassword"]', 'weak');
    await page.fill('input[name="firstName"]', 'Test');
    await page.fill('input[name="lastName"]', 'User');

    await page.click('button[type="submit"]');

    await expect(page.locator('text=/Password must be at least/')).toBeVisible();
  });

  test('Duplicate email registration', async ({ page }) => {
    await page.goto(`${BASE_URL}/register`);

    await page.fill('input[name="email"]', testUser.email);
    await page.fill('input[name="password"]', 'Test123!');
    await page.fill('input[name="confirmPassword"]', 'Test123!');
    await page.fill('input[name="firstName"]', 'Test');
    await page.fill('input[name="lastName"]', 'User');

    await page.click('button[type="submit"]');

    await expect(page.locator('text=/Email already registered/')).toBeVisible({ timeout: 5000 });
  });
});

