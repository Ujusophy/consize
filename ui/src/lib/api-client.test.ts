import { afterEach, describe, it, mock } from "node:test";
import assert from "node:assert/strict";
import { z } from "zod";
import { apiGet, ApiError } from "./api-client";
afterEach(() => mock.restoreAll());
const schema = z.object({ value: z.number() });
describe("API reads", () => {
  it("validates a response and forwards cancellation", async () => {
    const fetch = mock.method(globalThis, "fetch", (async () => new Response('{"value":0}')) as typeof globalThis.fetch);
    const controller = new AbortController();
    assert.deepEqual(await apiGet("/test", schema, controller.signal), { value: 0 });
    const args = fetch.mock.calls[0].arguments;
    assert.ok(String(args[0]).endsWith("/test"));
    assert.deepEqual(args[1], { signal: controller.signal, cache: "no-store" });
  });
  it("rejects malformed successful responses", async () => {
    mock.method(globalThis, "fetch", async () => new Response('{"value":"invalid"}'));
    await assert.rejects(apiGet("/test", schema), { status: -1 });
  });
  it("preserves permission refusal", async () => {
    mock.method(globalThis, "fetch", async () => new Response(
      JSON.stringify({ title: "Forbidden", status: 403 }), { status: 403 },
    ));
    await assert.rejects(apiGet("/test", schema), { status: 403, message: "Forbidden" });
  });
  it("reports network failure instead of inventing data", async () => {
    mock.method(globalThis, "fetch", async () => { throw new TypeError("connection refused"); });
    await assert.rejects(apiGet("/test", schema), (error: unknown) => error instanceof ApiError && error.status === 0);
  });
  it("preserves HTTP status for a non-JSON error", async () => {
    mock.method(globalThis, "fetch", async () => new Response("unavailable", { status: 503 }));
    await assert.rejects(apiGet("/test", schema), { status: 503 });
  });
});
