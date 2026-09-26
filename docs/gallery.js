// Concept-art gallery: filters by kind and a lightbox with prev/next, keys, and swipe.
// The DOM is built with createElement/textContent only. IMAGES is the one list of
// concept files in docs/assets; keep it in sync with assets/concept/*.jpg.
(function () {
  "use strict";

  var KINDS = [
    { id: "all", label: "All" },
    { id: "scene", label: "Scenes", one: "Scene" },
    { id: "combat", label: "Combat", one: "Combat" },
    { id: "battlemap", label: "Battle maps", one: "Battle map" },
    { id: "establishing", label: "Establishing", one: "Establishing" },
    { id: "ui", label: "UI", one: "UI" }
  ];

  // file: name in docs/assets; w/h: pixel size; alt: what the image shows.
  var IMAGES = [
    { file: "scene-tavern-barkeep-talk-rain.jpg", w: 1672, h: 941, subject: "tavern barkeep talk, rain",
      alt: "Concept art: two adventurers lean on a candlelit bar while a wary barkeep listens, rain on the harbor windows behind them." },
    { file: "scene-tavern-barkeep-talk-moonlit.jpg", w: 1672, h: 941, subject: "tavern barkeep talk, moonlit",
      alt: "Concept art: a ranger and an elf question a bearded barkeep across the bar, the moonlit harbor visible through the tavern windows." },
    { file: "scene-flooded-hall-party-ledge-statues.jpg", w: 1672, h: 941, subject: "flooded hall, party on a ledge, statues",
      alt: "Concept art: a party with torches walks a stone ledge above a flooded hall of giant statues lit by a blue glow." },
    { file: "scene-flooded-hall-party-wading-torchlit.jpg", w: 1672, h: 941, subject: "flooded hall, party wading, torchlit",
      alt: "Concept art: four adventurers wade through knee-deep water in a ruined temple, one holding a torch and one a glowing staff." },
    { file: "scene-crypt-lich-king-confrontation.jpg", w: 1672, h: 941, subject: "crypt, lich king confrontation",
      alt: "Concept art: a party with sword, axe, and spell faces a crowned lich glowing green on a crypt dais." },
    { file: "scene-library-arcane-tome-research.jpg", w: 1672, h: 941, subject: "library, arcane tome research",
      alt: "Concept art: two adventurers study a large open tome by candlelight in a towering library while an archer keeps watch." },
    { file: "scene-campfire-under-stars.jpg", w: 1672, h: 941, subject: "campfire under the stars",
      alt: "Concept art: a party rests around a campfire on a forest ridge under a starry sky, a lookout standing above a misty valley." },
    { file: "scene-swamp-lantern-trek-dusk.jpg", w: 1672, h: 941, subject: "swamp lantern trek, dusk",
      alt: "Concept art: a party led by a lantern-bearer picks its way through a mossy swamp at dusk." },
    { file: "scene-mountain-ruins-snowy-ridge-trek.jpg", w: 1672, h: 941, subject: "mountain ruins, snowy ridge trek",
      alt: "Concept art: cloaked adventurers climb a snowy ridge toward ruined towers above cloud-filled valleys." },
    { file: "scene-throne-room-royal-audience.jpg", w: 1672, h: 941, subject: "throne room, royal audience",
      alt: "Concept art: a party stands before a queen on her throne, flanked by armoured guards and red banners." },
    { file: "combat-harbor-docks-rain-close.jpg", w: 1672, h: 941, subject: "harbor docks, rain, close",
      alt: "Concept art: a close view of a rain-soaked dockside brawl, a shield fighter and a hooded rogue clashing with thugs among crates." },
    { file: "combat-harbor-docks-wide-lanterns.jpg", w: 1672, h: 941, subject: "harbor docks, wide, lanterns",
      alt: "Concept art: a wide view of a lantern-lit dock fight at night, a mage casting blue light while fighters engage raiders." },
    { file: "battlemap-tavern-flooded-common-room.jpg", w: 1672, h: 941, subject: "tavern, flooded common room",
      alt: "Top-down battle map concept: a tavern common room on a square grid, part of the floor flooded, with tables, stairs, and a moonlit window." },
    { file: "battlemap-harbor-docks-moonlit.jpg", w: 1672, h: 941, subject: "harbor docks, moonlit",
      alt: "Top-down battle map concept: moonlit harbor docks on a grid with crates, stairs, rowboats, and moored ships." },
    { file: "battlemap-flooded-hall-waterfall-temple.jpg", w: 1672, h: 941, subject: "flooded hall, waterfall temple",
      alt: "Top-down battle map concept: a flooded temple hall on a grid with waterfalls, broken bridges, and statues." },
    { file: "battlemap-crypt-sarcophagus-vault.jpg", w: 1672, h: 941, subject: "crypt, sarcophagus vault",
      alt: "Top-down battle map concept: a candlelit crypt vault on a grid with rows of sarcophagi and a raised altar." },
    { file: "battlemap-library-orrery-reading-hall.jpg", w: 1672, h: 941, subject: "library, orrery, reading hall",
      alt: "Top-down battle map concept: a library reading hall on a grid with desks, bookshelves, balconies, and a glowing orrery." },
    { file: "battlemap-throne-room-marble-hall.jpg", w: 1672, h: 941, subject: "throne room, marble hall",
      alt: "Top-down battle map concept: a marble throne room on a grid with a red carpet, banners, and candelabras leading to the throne." },
    { file: "battlemap-forest-ruins-standing-stones.jpg", w: 1672, h: 941, subject: "forest ruins, standing stones",
      alt: "Top-down battle map concept: overgrown forest ruins on a grid with a stone circle, broken walls, and a stream." },
    { file: "battlemap-swamp-boardwalk-shrine.jpg", w: 1672, h: 941, subject: "swamp boardwalk, shrine",
      alt: "Top-down battle map concept: wooden boardwalks crossing a dark swamp on a grid toward a candlelit shrine." },
    { file: "battlemap-mountain-ruins-cliff-bridges.jpg", w: 1672, h: 941, subject: "mountain ruins, cliff bridges",
      alt: "Top-down battle map concept: snowy mountain ruins on a grid joined by stone bridges over misty chasms." },
    { file: "battlemap-lava-bridges-fortress-gate.jpg", w: 1672, h: 941, subject: "lava bridges, fortress gate",
      alt: "Top-down battle map concept: stone walkways and chained bridges over rivers of lava leading to a fortress gate." },
    { file: "establishing-harbor-canal-rowboat-bridge.jpg", w: 1672, h: 941, subject: "harbor canal, rowboat, bridge",
      alt: "Establishing concept art: a lantern-lit rowboat crosses a harbor canal at night toward a stone bridge, a castle on the hill above." },
    { file: "establishing-harbor-cliff-castle-waterfalls.jpg", w: 1672, h: 941, subject: "harbor, cliff castle, waterfalls",
      alt: "Establishing concept art: a rainy harbor at night with sailing ships, below a clifftop castle and waterfalls under a full moon." },
    { file: "ui-tv-title-screen-join-lobby.jpg", w: 1672, h: 941, subject: "TV title screen, join lobby",
      alt: "Concept art of the TV title screen: a moonlit river town with the Drowned Lantern tavern lit along the water, a room code and QR code to join, and two example party members." },
    { file: "ui-tv-character-creation-phone-picker.jpg", w: 1672, h: 941, subject: "TV character creation, phone picker",
      alt: "Concept art of character creation: a painted portrait of a human fighter on the TV, engine-rolled ability scores in the centre, and a phone showing gender and species choices." },
    { file: "ui-tv-opening-scene-drowned-lantern-tavern.jpg", w: 1672, h: 941, subject: "TV opening scene, Drowned Lantern tavern",
      alt: "Concept art of the opening scene: a candlelit tavern interior with the party's portraits on the left and the dungeon master's narration across the bottom." },
    { file: "ui-tv-tavern-barkeep-dialogue-choices.jpg", w: 1672, h: 941, subject: "TV tavern barkeep dialogue, choices",
      alt: "Concept art of a conversation on the TV: two adventurers lean on the bar while the barkeep answers warily, with options to persuade her, ask something else, or look around." },
    { file: "ui-tv-sunken-halls-exploration-hud.jpg", w: 1672, h: 941, subject: "TV sunken halls exploration, HUD",
      alt: "Concept art of dungeon exploration on the TV: four adventurers with torches cross a flooded stone hall toward a statue, with party hit points on the left and the current objective on the right." },
    { file: "ui-phone-tavern-persuasion-sheet-screens.jpg", w: 1672, h: 941, subject: "phone screens: tavern, persuasion, sheet",
      alt: "Concept art of four phone screens: conversation replies, a Persuasion check rolling a 17 for a total of 21, a list of things to investigate, and a character sheet with combat actions." },
    { file: "ui-phone-sunken-halls-door-arcana-loot.jpg", w: 1670, h: 941, subject: "phone screens: sunken halls, door, arcana, loot",
      alt: "Concept art of four phone screens walking through exploration: a flooded hall, a strange door with check options, an Arcana check succeeding, and an amulet found in a chest." }
  ];

  // The kind comes from the file-name prefix (scene-, combat-, battlemap-, establishing-, ui-).
  function kindOf(file) {
    var prefix = file.split("-")[0];
    for (var i = 1; i < KINDS.length; i++) if (KINDS[i].id === prefix) return KINDS[i];
    return null;
  }

  // The subject is the file name after the prefix, as words:
  // "battlemap-lava-bridges-fortress-gate.jpg" -> "lava bridges fortress gate".
  // An entry's `subject` is the same words with commas where the phrases break.
  function subjectOf(img) {
    if (img.subject) return img.subject;
    return img.file.replace(/\.[a-z]+$/i, "").split("-").slice(1).map(function (w) {
      return /^(tv|hud|npc|dm)$/.test(w) ? w.toUpperCase() : w;
    }).join(" ");
  }

  // "Battle map: lava bridges, fortress gate".
  function captionOf(img) {
    var k = kindOf(img.file);
    return (k ? k.one + ": " : "") + subjectOf(img);
  }

  var grid = document.getElementById("gallery-grid");
  var chips = document.getElementById("gallery-filters");
  var status = document.getElementById("gallery-status");
  var box = document.getElementById("gallery-box");
  if (!grid || !chips || !box) return;

  var items = IMAGES.map(function (img, i) {
    var k = kindOf(img.file);
    return { img: img, index: i, kind: k ? k.id : "other", caption: captionOf(img), el: null, link: null };
  });
  var visible = items.slice();
  var current = -1;
  var opener = null;

  // ---------- Grid ----------

  items.forEach(function (it) {
    var fig = document.createElement("figure");
    fig.className = "frame gallery-item";
    fig.dataset.kind = it.kind;

    var a = document.createElement("a");
    a.className = "gallery-link";
    a.href = "assets/" + it.img.file;

    var im = document.createElement("img");
    im.src = "assets/" + it.img.file;
    im.width = it.img.w;
    im.height = it.img.h;
    im.loading = "lazy";
    im.decoding = "async";
    im.alt = it.img.alt;
    a.appendChild(im);

    var cap = document.createElement("figcaption");
    var tag = document.createElement("span");
    tag.className = "gallery-kind";
    tag.textContent = (kindOf(it.img.file) || { one: "Concept" }).one;
    var sub = document.createElement("span");
    sub.className = "gallery-subject";
    sub.textContent = subjectOf(it.img);
    var sep = document.createElement("span");
    sep.className = "visually-hidden";
    sep.textContent = ": ";
    cap.appendChild(tag);
    cap.appendChild(sep);
    cap.appendChild(sub);

    fig.appendChild(a);
    fig.appendChild(cap);
    grid.appendChild(fig);

    a.addEventListener("click", function (e) {
      if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button === 1) return;
      if (typeof box.showModal !== "function") return;
      e.preventDefault();
      opener = a;
      open(visible.indexOf(it));
    });
    it.el = fig;
    it.link = a;
  });

  // ---------- Filters ----------

  var counts = { all: items.length };
  items.forEach(function (it) { counts[it.kind] = (counts[it.kind] || 0) + 1; });

  KINDS.forEach(function (k) {
    if (k.id !== "all" && !counts[k.id]) return;
    var b = document.createElement("button");
    b.type = "button";
    b.className = "chip" + (k.id === "all" ? " is-on" : "");
    b.dataset.filter = k.id;
    b.setAttribute("aria-pressed", k.id === "all" ? "true" : "false");
    b.appendChild(document.createTextNode(k.label + " "));
    var n = document.createElement("span");
    n.className = "chip-n";
    n.textContent = String(counts[k.id] || 0);
    b.appendChild(n);
    chips.appendChild(b);
  });
  chips.hidden = false;

  function applyFilter(f) {
    chips.querySelectorAll(".chip").forEach(function (c) {
      var on = c.dataset.filter === f;
      c.classList.toggle("is-on", on);
      c.setAttribute("aria-pressed", on ? "true" : "false");
    });
    visible = items.filter(function (it) { return f === "all" || it.kind === f; });
    items.forEach(function (it) { it.el.hidden = visible.indexOf(it) === -1; });
    if (status) status.textContent = "Showing " + visible.length + " of " + items.length + " images.";
  }

  chips.addEventListener("click", function (e) {
    var chip = e.target.closest(".chip");
    if (chip) applyFilter(chip.dataset.filter);
  });
  applyFilter("all");

  // ---------- Lightbox ----------

  var boxImg = box.querySelector(".gallery-box-img");
  var boxCap = box.querySelector(".gallery-box-caption");
  var boxCount = box.querySelector(".gallery-box-count");
  var boxOrig = box.querySelector(".gallery-box-original");
  var prevBtn = box.querySelector(".gallery-prev");
  var nextBtn = box.querySelector(".gallery-next");
  var closeBtn = box.querySelector(".gallery-close");

  function show(i) {
    var n = visible.length;
    if (!n) return;
    current = (i + n) % n;
    var it = visible[current];
    boxImg.src = "assets/" + it.img.file;
    boxImg.width = it.img.w;
    boxImg.height = it.img.h;
    boxImg.alt = it.img.alt;
    boxCap.textContent = it.caption;
    boxCount.textContent = (current + 1) + " of " + n;
    boxOrig.href = "assets/" + it.img.file;
    var single = n < 2;
    prevBtn.hidden = single;
    nextBtn.hidden = single;
    // Warm the neighbours so prev/next feel instant.
    [current - 1, current + 1].forEach(function (j) {
      var nb = visible[(j + n) % n];
      var pre = new Image();
      pre.src = "assets/" + nb.img.file;
    });
  }

  function open(i) {
    show(i < 0 ? 0 : i);
    box.showModal();
    closeBtn.focus();
  }

  prevBtn.addEventListener("click", function () { show(current - 1); });
  nextBtn.addEventListener("click", function () { show(current + 1); });
  // Close and hand focus back to the thumbnail that opened the lightbox.
  // Done synchronously here; the "close" event covers any other way the dialog closes.
  function closeBox() {
    if (box.open) box.close();
    restore();
  }
  function restore() {
    if (!opener) return;
    var o = opener;
    opener = null;
    boxImg.removeAttribute("src");
    o.focus();
  }

  closeBtn.addEventListener("click", closeBox);

  box.addEventListener("keydown", function (e) {
    if (e.key === "ArrowLeft") { e.preventDefault(); show(current - 1); }
    else if (e.key === "ArrowRight") { e.preventDefault(); show(current + 1); }
    else if (e.key === "Escape") { e.preventDefault(); closeBox(); }
    else if (e.key === "Tab") trapTab(e);
  });

  // The modal dialog already makes the page inert; this keeps Tab cycling
  // inside the dialog instead of leaving for the browser chrome.
  function trapTab(e) {
    var f = Array.prototype.filter.call(
      box.querySelectorAll("button, a[href]"),
      function (el) { return !el.hidden && el.offsetParent !== null; }
    );
    if (!f.length) return;
    var first = f[0], last = f[f.length - 1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
  }

  // Click on the backdrop closes.
  box.addEventListener("click", function (e) { if (e.target === box) closeBox(); });

  box.addEventListener("close", restore);

  // Swipe left/right on touch.
  var sx = 0, sy = 0, tracking = false;
  box.addEventListener("touchstart", function (e) {
    if (e.touches.length !== 1) { tracking = false; return; }
    tracking = true;
    sx = e.touches[0].clientX;
    sy = e.touches[0].clientY;
  }, { passive: true });
  box.addEventListener("touchend", function (e) {
    if (!tracking) return;
    tracking = false;
    var t = e.changedTouches[0];
    var dx = t.clientX - sx, dy = t.clientY - sy;
    if (Math.abs(dx) > 40 && Math.abs(dx) > Math.abs(dy) * 1.5) show(current + (dx < 0 ? 1 : -1));
  }, { passive: true });
})();
