import { expect, test } from "@playwright/test";
import { base, openAuthenticated } from "./fixtures/mock-api";

test("sidebar and content never overlap", async ({ page }) => {
  await openAuthenticated(page, `${base}/catalog/products`);
  const sidebar = page.locator('[data-slot="sidebar-container"]');
  const inset = page.locator('[data-slot="sidebar-inset"]');
  const sidebarBox = await sidebar.boundingBox();
  const insetBox = await inset.boundingBox();
  expect(sidebarBox).not.toBeNull();
  expect(insetBox).not.toBeNull();
  expect(insetBox!.x).toBeGreaterThanOrEqual(sidebarBox!.x + sidebarBox!.width - 2);
});

test("mobile sidebar opens as a sheet", async ({ page }) => {
  await openAuthenticated(page);
  await page.getByRole("button", { name: "Toggle Sidebar" }).click();
  await expect(page.getByText("Billing console")).toBeVisible();
});

test("editor actions remain reachable", async ({ page }) => {
  await openAuthenticated(page, `${base}/subscriptions/new`);
  await page.getByRole("button", { name: "Save subscription" }).scrollIntoViewIfNeeded();
  await expect(page.getByRole("button", { name: "Save subscription" })).toBeVisible();
});

test("data table toolbar has consistent spacing without a nested card", async ({ page }) => {
  await openAuthenticated(page, `${base}/usage`);
  const search = page.getByPlaceholder("Search event ID…");
  const table = page.locator('[data-slot="table-container"]');
  const listSurface = page.locator(".resource-list");
  const searchBox = await search.boundingBox();
  const tableBox = await table.boundingBox();
  expect(searchBox).not.toBeNull();
  expect(tableBox).not.toBeNull();
  expect(Math.round(tableBox!.y - (searchBox!.y + searchBox!.height))).toBe(16);
  await expect(listSurface).toHaveCSS("border-top-width", "0px");
  await expect(listSurface).toHaveCSS("box-shadow", "none");
});

test("nested sidebar menus can be opened and closed", async ({ page }) => {
  await openAuthenticated(page);
  const catalog = page.getByRole("button", { name: "Catalog" });
  await catalog.click();
  await expect(page.getByRole("link", { name: "Products" })).toBeVisible();
  await catalog.click();
  await expect(page.getByRole("link", { name: "Products" })).toBeHidden();
});
