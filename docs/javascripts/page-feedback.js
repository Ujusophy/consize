document.addEventListener("click", function (event) {
  var btn = event.target.closest(".consize-page-feedback__btn");
  if (!btn) return;

  var group = btn.closest("[data-consize-feedback]");
  if (!group) return;

  group.innerHTML = '<span class="consize-page-feedback__thanks">Thanks for your feedback!</span>';
});