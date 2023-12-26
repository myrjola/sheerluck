import { test, expect } from '@playwright/test';

test('Webauthn register, log out and sign in', async ({ page }) => {
  await page.goto('/');

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle(/Sheerluck/);

  // Configure webauthn https://github.com/microsoft/playwright/issues/7276#issuecomment-1516768428
  const cdpSession = await page.context().newCDPSession(page);
  await cdpSession.send('WebAuthn.enable');
  await cdpSession.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      transport: 'internal',
      hasUserVerification: true,
      isUserVerified: true,
      hasResidentKey: true,
    },
  });

  await page.getByRole('button', { name: 'Register' }).click();
  await page.getByRole('button', { name: 'Log out' }).click();
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByRole('button', { name: 'Log out' })).toBeVisible();
});