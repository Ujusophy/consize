document.addEventListener("DOMContentLoaded", function () {
  // Curated links shown before the user types anything —
  // mirrors Merge's docs search: click in, see useful destinations immediately.
  var suggestions = [
    { title: "Get started", desc: "Go from zero to a verified rightsizing change.", url: "getting-started/get-started/" },
    { title: "Interactive Sandbox", desc: "Try Consize locally in one Docker container.", url: "getting-started/sandbox/" },
    { title: "Production Installation", desc: "Install the Helm chart onto a live cluster.", url: "getting-started/installation/" },
    { title: "Kubernetes Rightsizing", desc: "How CPU/memory requests and limits get resized.", url: "guides/rightsizing/" },
    { title: "How Consize Works", desc: "The full Observe → Analyze → Verify architecture.", url: "concepts/architecture/" },
    { title: "Configuration", desc: "Full reference for what Consize can observe and change.", url: "reference/configuration/" }
  ];

  var input = document.querySelector(".md-search__input");
  var scrollwrap = document.querySelector(".md-search__scrollwrap");
  var result = document.querySelector(".md-search-result");
  if (!input || !scrollwrap || !result) return;

  // Material exposes the page's relative base path in #__config so links
  // resolve correctly no matter how deep the current page is nested.
  var base = ".";
  try {
    base = JSON.parse(document.getElementById("__config").textContent).base;
  } catch (e) {}

  var panel = document.createElement("div");
  panel.className = "consize-search-suggestions";
  panel.innerHTML =
    '<div class="consize-search-suggestions__label">Suggested</div>' +
    '<ul class="consize-search-suggestions__list">' +
    suggestions.map(function (s) {
      return (
        '<li><a class="consize-search-suggestions__link" href="' + base + "/" + s.url + '">' +
          '<span class="consize-search-suggestions__title">' + s.title + '</span>' +
          '<span class="consize-search-suggestions__desc">' + s.desc + '</span>' +
        '</a></li>'
      );
    }).join("") +
    '</ul>';

  scrollwrap.insertBefore(panel, result);

  function sync() {
    var empty = input.value.trim().length === 0;
    panel.style.display = empty ? "" : "none";
    result.style.display = empty ? "none" : "";
  }

  input.addEventListener("focus", sync);
  input.addEventListener("input", sync);
  sync();
});