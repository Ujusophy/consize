document.addEventListener("click", function (event) {
  var btn = event.target.closest(".consize-page-feedback__btn");
  if (!btn) return;

  var group = btn.closest("[data-consize-feedback]");
  if (!group) return;

  group.innerHTML = '<span class="consize-page-feedback__thanks">Thanks for your feedback!</span>';
});

document.addEventListener("click", function (e) {
  var btn = e.target.closest(
    '.md-clipboard, .md-code__button[data-md-type="copy"], [data-clipboard-target], [data-clipboard-text]'
  );
  if (!btn || btn.classList.contains("consize-copied")) return;

  var label = document.createElement("span");
  label.className = "consize-copied__label";
  label.textContent = "Copied!";
  btn.appendChild(label);
  btn.classList.add("consize-copied");
  btn.blur();

  setTimeout(function () {
    label.remove();
    btn.classList.remove("consize-copied");
  }, 2000);
});