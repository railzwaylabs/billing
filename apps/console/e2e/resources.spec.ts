import { expect, test } from "@playwright/test";
import { base, mockAPI, openAuthenticated } from "./fixtures/mock-api";

const listPages = [
  ["Meters", "meters"], ["Products", "catalog/products"], ["Prices", "catalog/prices"], ["Subscriptions", "subscriptions"],
  ["Usage events", "usage"], ["Customers", "customers"], ["Invoices", "invoices"], ["Service accounts", "developer/service-accounts"],
  ["API keys", "developer/api-keys"], ["Roles", "iam/roles"], ["Policy", "iam/policy"], ["Organization settings", "settings"],
] as const;

for (const [heading, path] of listPages) {
  test(`opens ${heading}`, async ({ page }) => {
    await openAuthenticated(page, `${base}/${path}`);
    await expect(page.getByRole("heading", { name: heading })).toBeVisible();
  });
}

test("creates a meter with a custom usage unit", async ({ page }) => {
  let payload: Record<string, unknown> | undefined;
  await mockAPI(page);
  await page.route("**/admin/v1/organizations/*/meters", async (route) => {
    if (route.request().method() === "POST") {
      payload = route.request().postDataJSON();
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ meter: { ID: "meter-1", ...payload } }) });
    }
    return route.fallback();
  });
  await page.goto(`${base}/meters/new`);
  const inputs = page.locator("form input");
  await inputs.nth(0).fill("API_CALLS");
  await inputs.nth(1).fill("API calls");
  await page.getByPlaceholder("Search or enter a usage unit…").fill("request");
  await page.keyboard.press("Enter");
  await page.getByRole("button", { name: "Save meter" }).click();
  expect(payload).toMatchObject({ code: "API_CALLS", name: "API calls", unit: "request", aggregation: "sum" });
});

test("client validation prevents an incomplete meter", async ({ page }) => {
  await openAuthenticated(page, `${base}/meters/new`);
  await expect(page.getByRole("button", { name: "Save meter" })).toBeDisabled();
});

const failures = [
  [401, "UNAUTHENTICATED", "Authentication required"], [403, "PERMISSION_DENIED", "Permission denied"], [404, "RESOURCE_NOT_FOUND", "Resource not found"],
  [409, "RESOURCE_CONFLICT", "Resource already exists"], [422, "VALIDATION_FAILED", "Invalid resource"], [500, "INTERNAL", "Unable to save resource"],
] as const;

for (const [status, code, message] of failures) {
  test(`surfaces ${status} ${code} from resource mutation`, async ({ page }) => {
    await mockAPI(page, { failure: { method: "POST", path: /\/customers$/, status, code, message } });
    await page.goto(`${base}/customers/new`);
    const inputs = page.locator("form input");
    await inputs.nth(0).fill("Ada");
    await inputs.nth(1).fill("Lovelace");
    await page.getByRole("button", { name: "Save customer" }).click();
    await expect(page.getByText(message)).toBeVisible();
  });
}
