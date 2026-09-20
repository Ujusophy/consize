/* Theme Mermaid diagrams to match the Consize brand palette instead of
   Mermaid's stock default theme, and keep them in sync with the light/dark
   toggle (re-runs whenever MkDocs Material's instant navigation swaps
   pages, and whenever the color scheme changes). */

function consizeMermaidTheme() {
  const isDark = document.body.getAttribute("data-md-color-scheme") === "slate";

  return isDark
    ? {
        background: "#0c0e0d",
        primaryColor: "#0f3d2e",
        primaryTextColor: "#e8fdf1",
        primaryBorderColor: "#34d399",
        lineColor: "#34d399",
        secondaryColor: "#131615",
        tertiaryColor: "#0c0e0d",
        fontFamily: "Inter, sans-serif",
        fontSize: "15px",
      }
    : {
        primaryColor: "#e3f5ed",
        primaryTextColor: "#06382a",
        primaryBorderColor: "#0b8961",
        lineColor: "#0b8961",
        secondaryColor: "#f2faf6",
        tertiaryColor: "#ffffff",
        fontFamily: "Inter, sans-serif",
        fontSize: "15px",
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