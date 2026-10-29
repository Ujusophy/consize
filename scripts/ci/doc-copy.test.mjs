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
