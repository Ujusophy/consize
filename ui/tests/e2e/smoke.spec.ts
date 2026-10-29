import { expect, test } from "../../../tools/ci/node_modules/@playwright/test/index.js";

test("production UI reads the real Go API without mocks", async ({ page, request }) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  const api = "http://127.0.0.1:18080";
  expect((await request.get(`${api}/api/health`)).ok()).toBeTruthy();
  const summaryResponse = page.waitForResponse(response =>
    response.url().startsWith(`${api}/api/v1/dashboard/summary`) && response.ok());
  const recommendationsResponse = page.waitForResponse(response =>
    response.url().startsWith(`${api}/api/v1/recommendations`) && response.ok());
  await page.goto("/");
  const summary = await (await summaryResponse).json();
  const recommendations = await (await recommendationsResponse).json();
  expect(summary).toHaveProperty("open_recommendations", 0);
  expect(recommendations.items).toEqual([]);
  await expect(page.getByRole("heading", { name: "Prioritized recommendations" })).toBeVisible();
  await expect(page.getByText("Evidence is being collected.", { exact: false })).toBeVisible();
  await page.getByRole("navigation", { name: "Primary" }).getByRole("link", { name: "Recommendations", exact: true }).click();
  await expect(page).toHaveURL(/\/recommendations/);
  await expect(page.getByRole("heading", { name: "Recommendations", exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("heading", { name: "Recommendations", exact: true })).toBeVisible();
  expect(await page.evaluate(async () => (await navigator.serviceWorker.getRegistrations()).length)).toBe(0);
  expect(errors).toEqual([]);
});
