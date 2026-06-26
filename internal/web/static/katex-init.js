// Render LaTeX math in card text. The only JavaScript in the viewer: a static,
// self-hosted KaTeX auto-render pass over the page body once it loads.
document.addEventListener("DOMContentLoaded", function () {
  if (typeof window.renderMathInElement !== "function") return;
  renderMathInElement(document.body, {
    delimiters: [
      { left: "$$", right: "$$", display: true },
      { left: "$", right: "$", display: false }
    ],
    ignoredTags: ["script", "noscript", "style", "textarea", "pre", "code"],
    throwOnError: false
  });
});
