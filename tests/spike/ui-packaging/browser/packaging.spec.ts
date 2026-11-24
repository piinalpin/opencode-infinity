import { readFileSync } from 'node:fs';
import { test, expect } from '@playwright/test';

const origin = process.env.T05_ORIGIN!;
const password = readFileSync(process.env.T05_ADMIN_SECRET_FILE!, 'utf8');

test('mount dan aset lokal', async ({ browser }) => {
  const context = await browser.newContext({ httpCredentials: { username: 'admin', password }, serviceWorkers: 'block' });
  const page = await context.newPage();
  const failures: string[] = [];
  const assets: string[] = [];
  page.on('pageerror', () => failures.push('pageerror'));
  page.on('console', message => { if (message.type() === 'error') failures.push('console_error'); });
  page.on('requestfailed', () => failures.push('requestfailed'));
  await context.route('**/*', async route => {
    if (new URL(route.request().url()).origin !== origin) {
      failures.push('external_request');
      await route.abort();
    } else { await route.continue(); }
  });
  page.on('response', response => {
    const path = new URL(response.url()).pathname;
    if (path.startsWith('/assets/')) {
      if (response.status() !== 200) failures.push('asset_status');
      assets.push(path);
    }
  });
  await page.goto(origin);
  await expect(page.getByRole('heading', { name: 'Fixture kemasan T05' })).toBeVisible();
  await expect(page.getByText('Data sintetis. Bukan dashboard pemantauan pool.')).toBeVisible();
  await page.getByRole('button', { name: 'Uji interaksi lokal' }).click();
  await expect(page.getByRole('status')).toHaveText('Interaksi lokal: 1');
  await expect(page.locator('main')).toHaveCSS('max-width', '640px');
  expect(assets.some(path => path.endsWith('.js'))).toBeTruthy();
  expect(assets.some(path => path.endsWith('.css'))).toBeTruthy();
  expect(failures).toEqual([]);
  const storage = await page.evaluate(() => ({ local: localStorage.length, session: sessionStorage.length }));
  expect(storage).toEqual({ local: 0, session: 0 });
  await context.close();
});

test('browser tanpa credential ditolak', async ({ browser }) => {
  const context = await browser.newContext();
  const page = await context.newPage();
  const response = await page.goto(origin).catch(() => null);
  if (response) expect(response.status()).toBe(401);
  else {
    // Chromium dapat menolak navigasi challenge HTTP tanpa credential.
    const check = await context.request.get(origin);
    expect(check.status()).toBe(401);
  }
  await expect(page.getByRole('heading', { name: 'Fixture kemasan T05' })).toHaveCount(0);
  await context.close();
});
