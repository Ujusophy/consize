/* Adds hover/focus dropdown panels to the top nav tabs. Material's
   navigation.tabs feature only renders a tab bar — clicking a tab
   navigates straight to its first child page rather than showing a
   menu. This mirrors the "nav:" tree in mkdocs.yml as a dropdown.

   IMPORTANT: keep this in sync with mkdocs.yml's nav: block by hand —
   it's a plain JS object, not read from the YAML. */

const CONSIZE_NAV_MENUS = {
  "Get started": [
    ["Get started", "getting-started/get-started/"],
    ["Interactive Sandbox", "getting-started/sandbox/"],
    ["Production Installation", "getting-started/installation/"],
  ],
  Product: [
    ["Kubernetes Rightsizing", "guides/rightsizing/"],
    ["Observability", "guides/observability/"],
    ["Environments", "guides/environments/"],
  ],
  Platform: [
    ["How Consize Works", "concepts/architecture/"],
    ["The Safety Net", "concepts/safety-net/"],
  ],
  Reference: [["Configuration", "reference/configuration/"]],
  Community: [
    ["Contributing", "contributing/"],
    ["Social channels & blog", "https://discord.gg/REPLACE_ME"],
    ["Decisions", "contributing/decisions/"],
    ["Roadmap", "resources/roadmap/"],
  ],
};

document$.subscribe(() => {
  const base = document.querySelector("base")?.getAttribute("href") || "/";

  document.querySelectorAll(".md-tabs__item").forEach((tab) => {
    tab.querySelector(".consize-tab-dropdown")?.remove();

    const link = tab.querySelector(".md-tabs__link");
    const items = link && CONSIZE_NAV_MENUS[link.textContent.trim()];
    if (!items) return;

    const menu = document.createElement("div");
    menu.className = "consize-tab-dropdown";

    items.forEach(([title, href]) => {
      const a = document.createElement("a");
      a.textContent = title;
      a.href = href.startsWith("http") ? href : base + href;
      menu.appendChild(a);
    });

    tab.appendChild(menu);
  });
});