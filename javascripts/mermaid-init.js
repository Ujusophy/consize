/* Theme Mermaid diagrams to match the Consize brand palette instead of
   Mermaid's stock default theme, and keep them in sync with the light/dark
   toggle (re-runs whenever MkDocs Material's instant navigation swaps
   pages, and whenever the color scheme changes). */

function consizeMermaidTheme() {
  const isDark = document.body.getAttribute("data-md-color-scheme") === "slate";

  return isDark
    ? {
        background: "#101110",
        primaryColor: "#14532d",
        primaryTextColor: "#e8fdf1",
        primaryBorderColor: "#4ade80",
        lineColor: "#4ade80",
        secondaryColor: "#1a1b1a",
        tertiaryColor: "#101110",
        fontFamily: "Inter, sans-serif",
      }
    : {
        primaryColor: "#dcfce7",
        primaryTextColor: "#14532d",
        primaryBorderColor: "#16a34a",
        lineColor: "#16a34a",
        secondaryColor: "#f0fdf4",
        tertiaryColor: "#ffffff",
        fontFamily: "Inter, sans-serif",
      };
}

document$.subscribe(() => {
  if (typeof mermaid === "undefined") return;

  mermaid.initialize({
    startOnLoad: true,
    theme: "base",
    themeVariables: consizeMermaidTheme(),
  });

  mermaid.run({ querySelector: ".mermaid" });
});

/* Re-theme immediately when the user flips the light/dark toggle, without
   waiting for a full page navigation. */
document.addEventListener("click", (event) => {
  if (event.target.closest("[data-md-color-scheme]") || event.target.closest(".md-header__button")) {
    setTimeout(() => {
      if (typeof mermaid === "undefined") return;
      mermaid.initialize({ startOnLoad: true, theme: "base", themeVariables: consizeMermaidTheme() });
      mermaid.run({ querySelector: ".mermaid" });
    }, 50);
  }
});