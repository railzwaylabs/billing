import { expect, test } from "@playwright/test";
import { base, openAuthenticated } from "./fixtures/mock-api";

test("developer monitor shows resource allocation bar charts", async ({ page }) => {
  await openAuthenticated(page, `${base}/developer/monitor`);

  await expect(page.getByRole("heading", { name: "Monitor", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Resource allocation" })).toBeVisible();
  await expect(page.getByText("CPU allocation", { exact: true })).toBeVisible();
  await expect(page.getByText("Memory allocation", { exact: true })).toBeVisible();
  await expect(page.getByText("Disk allocation", { exact: true })).toBeVisible();
  await expect(page.getByText("Network throughput", { exact: true })).toBeVisible();
  await expect(page.locator(".recharts-bar")).toHaveCount(8);
  await expect(page.getByRole("tab", { name: "Daily" })).toHaveAttribute("data-state", "active");
  await page.getByRole("tab", { name: "Weekly" }).click();
  await expect(page.getByRole("tab", { name: "Weekly" })).toHaveAttribute("data-state", "active");
  await expect(page.getByText("Live", { exact: true })).toBeVisible();
});
