import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
const source = readFileSync(new URL("../../docs/javascripts/copy-page.js", import.meta.url), "utf8");

for (const [pathname, expected] of [["/guide/index.html", "/guide/index.md"], ["/guide/", "/guide/index.md"], ["/guide", "/guide/index.md"], ["/index.htmlindex.html", "/index.htmlindex.html/index.md"]]) {
  test(`copy page URL for ${pathname}`, async () => {
    let mounted;
    let fetched;
    const node = () => ({ children: [], setAttribute() {}, append(...children) { this.children.push(...children); }, addEventListener(type, fn) { this[type] = fn; }, replaceWith() {} });
    const toc = { append(value) { mounted = value; } };
    const context = {
      location: { pathname, origin: "https://docs.example.test" },
      document: {
        readyState: "complete", createElement: node, createElementNS: node,
        querySelectorAll: () => [], querySelector: selector => selector.includes("sidebar") ? toc : {},
      },
      navigator: { clipboard: { async writeText(text) { assert.equal(text, "synthetic markdown"); } } },
      window: {}, setTimeout() {},
      async fetch(url) { fetched = url; return { ok: true, headers: { get: () => "text/plain" }, text: async () => "synthetic markdown" }; },
    };
    vm.runInNewContext(source, context);
    const button = mounted.children[0];
    button.firstChild = button.children[0];
    await button.click();
    assert.equal(fetched, `https://docs.example.test${expected}`);
  });
}

test("copied Markdown table escapes both backslashes and pipes", async () => {
  let mounted;
  let copied;
  const node = () => ({ children: [], setAttribute() {}, append(...children) { this.children.push(...children); }, addEventListener(type, fn) { this[type] = fn; }, replaceWith() {} });
  const cell = { nodeType: 3, textContent: String.raw`a\|b` };
  const root = {
    nodeType: 1, tagName: "TABLE", className: "", cloneNode() { return this; },
    querySelectorAll: selector => selector === "tr" ? [{ children: [cell] }] : [],
  };
  vm.runInNewContext(source, {
    location: { pathname: "/guide/", origin: "https://docs.example.test" },
    document: {
      readyState: "complete", createElement: node, createElementNS: node, querySelectorAll: () => [],
      querySelector: selector => selector.includes("sidebar") ? { append(value) { mounted = value; } } : root,
    },
    navigator: { clipboard: { async writeText(text) { copied = text; } } },
    window: {}, setTimeout() {}, async fetch() { return { ok: false }; },
  });
  const button = mounted.children[0];
  button.firstChild = button.children[0];
  await button.click();
  assert.ok(copied.includes(String.raw`a\\\|b`), copied);
});
