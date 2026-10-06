---
title: Blog
hide:
  - navigation
  - toc
---

# The Consize Blog

Engineering write-ups, product updates, and honest takes on making cost optimization safe enough to actually turn on.

[Try the sandbox](../getting-started/sandbox.md){ .md-button .md-button--primary }
[Documentation](../getting-started/get-started.md){ .md-button }

<div class="blog-filter" role="toolbar" aria-label="Filter blog posts">
  <div class="blog-filter__tabs">
    <button type="button" class="blog-filter__tab is-active" data-filter="all">All</button>
    <button type="button" class="blog-filter__tab" data-filter="engineering">Engineering</button>
    <button type="button" class="blog-filter__tab" data-filter="finops">FinOps</button>
    <button type="button" class="blog-filter__tab" data-filter="community">Community</button>
    <button type="button" class="blog-filter__tab" data-filter="product">Product</button>
    <button type="button" class="blog-filter__tab" data-filter="security">Security</button>
    <button type="button" class="blog-filter__tab" data-filter="announcements">Announcements</button>
  </div>
  <label class="blog-filter__search">
    <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path fill="currentColor" d="M9.5 3A6.5 6.5 0 0 1 16 9.5c0 1.61-.59 3.09-1.56 4.23l.27.27h.79l5 5-1.5 1.5-5-5v-.79l-.27-.27A6.52 6.52 0 0 1 9.5 16 6.5 6.5 0 0 1 3 9.5 6.5 6.5 0 0 1 9.5 3m0 2C7 5 5 7 5 9.5S7 14 9.5 14 14 12 14 9.5 12 5 9.5 5"/></svg>
    <input type="search" id="blog-search" placeholder="Search..." autocomplete="off" aria-label="Search posts">
  </label>
</div>
<p class="blog-filter__empty" style="display:none">No posts match your filters.</p>
<script>
(function () {
  if (window.__blogFilterBound) return;
  window.__blogFilterBound = true;
  function slugs(post) {
    return Array.prototype.map.call(
      post.querySelectorAll('.md-post__meta a[href*="/category/"]'),
      function (a) { return a.getAttribute("href").replace(/\/$/, "").split("/").pop().toLowerCase(); }
    );
  }
  function apply() {
    var active = document.querySelector(".blog-filter__tab.is-active");
    var input = document.getElementById("blog-search");
    if (!active || !input) return;
    var filter = active.getAttribute("data-filter");
    var query = input.value.trim().toLowerCase();
    var shown = 0;
    document.querySelectorAll(".md-post").forEach(function (post) {
      var okCat = filter === "all" || slugs(post).indexOf(filter) !== -1;
      var okQuery = !query || post.textContent.toLowerCase().indexOf(query) !== -1;
      var show = okCat && okQuery;
      post.style.display = show ? "" : "none";
      if (show) shown++;
    });
    var empty = document.querySelector(".blog-filter__empty");
    if (empty) empty.style.display = shown ? "none" : "";
  }
  document.addEventListener("click", function (e) {
    var tab = e.target.closest(".blog-filter__tab");
    if (!tab) return;
    document.querySelectorAll(".blog-filter__tab").forEach(function (t) { t.classList.remove("is-active"); });
    tab.classList.add("is-active");
    apply();
  });
  document.addEventListener("input", function (e) {
    if (e.target.id === "blog-search") apply();
  });
})();
</script>