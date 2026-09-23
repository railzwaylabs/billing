import { expect, test } from "@playwright/test";
import { base, mockAPI, openAuthenticated, organization } from "./fixtures/mock-api";

test("restores an authenticated session and opens the organization", async ({ page }) => {
  await openAuthenticated(page);
  await expect(page).toHaveURL(new RegExp(`${base}$`));
  await expect(page.getByRole("heading", { name: organization.name })).toBeVisible();
});

test("logs in with local credentials", async ({ page }) => {
  await mockAPI(page, { authenticated: false });
  await page.goto("/");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(new RegExp(`${base}$`));
});

test("shows invalid credential error", async ({ page }) => {
  await mockAPI(page, { authenticated: false, failure: { method: "POST", path: /\/auth\/login$/, status: 401, code: "UNAUTHENTICATED", message: "Invalid username or password" } });
  await page.goto("/");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByText("Invalid username or password")).toBeVisible();
});

test("expired session returns to login", async ({ page }) => {
  await mockAPI(page, { authenticated: false });
  await page.goto(base);
  await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible();
});
