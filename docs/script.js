// Mobile nav toggle and a concept-art lightbox. The page works without this file.
(function () {
  document.documentElement.classList.remove("no-js");

  var toggle = document.querySelector(".nav-toggle");
  var nav = document.getElementById("site-nav");
  if (toggle && nav) {
    var setOpen = function (open) {
      toggle.setAttribute("aria-expanded", String(open));
      nav.classList.toggle("open", open);
    };
    toggle.addEventListener("click", function () {
      setOpen(toggle.getAttribute("aria-expanded") !== "true");
    });
    nav.addEventListener("click", function (e) {
      if (e.target.closest("a")) setOpen(false);
    });
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && nav.classList.contains("open")) {
        setOpen(false);
        toggle.focus();
      }
    });
  }

  var box = document.querySelector(".lightbox");
  if (!box || typeof box.showModal !== "function") return;
  var boxImg = box.querySelector("img");
  var boxCap = box.querySelector(".lightbox-caption");
  var opener = null;

  document.querySelectorAll("a.zoom").forEach(function (link) {
    link.addEventListener("click", function (e) {
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.button === 1) return;
      e.preventDefault();
      var img = link.querySelector("img");
      opener = link;
      boxImg.src = link.getAttribute("href");
      boxImg.alt = img ? img.alt : "";
      boxCap.textContent = (link.getAttribute("data-caption") || "") + " Concept art.";
      box.showModal();
    });
  });
  box.addEventListener("click", function (e) {
    if (e.target === box) box.close();
  });
  box.addEventListener("close", function () {
    if (opener) opener.focus();
  });
})();
