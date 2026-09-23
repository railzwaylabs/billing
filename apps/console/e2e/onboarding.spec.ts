import { expect, test } from "@playwright/test";
import { mockAPI, organization } from "./fixtures/mock-api";

test("creates the first organization", async ({ page }) => {
  await mockAPI(page, { organizations: [] });
  await page.goto("/");
  await page.getByPlaceholder("Acme, Inc.").fill("Acme");
  await page.getByRole("button", { name: "Create organization" }).click();
  await expect(page.getByText(organization.name).first()).toBeVisible();
});

test("requires organization name and slug", async ({ page }) => {
  await mockAPI(page, { organizations: [] });
  await page.goto("/");
  await page.getByRole("button", { name: "Create organization" }).click();
  await expect(page.getByPlaceholder("Acme, Inc.")).toHaveJSProperty("validity.valid", false);
});

test("shows duplicate slug conflict", async ({ page }) => {
  await mockAPI(page, { organizations: [], failure: { method: "POST", path: /\/organizations$/, status: 409, code: "ORGANIZATION_CONFLICT", message: "Organization slug already exists" } });
  await page.goto("/");
  await page.getByPlaceholder("Acme, Inc.").fill("Acme");
  await page.getByRole("button", { name: "Create organization" }).click();
  await expect(page.getByText("Organization slug already exists")).toBeVisible();
});
