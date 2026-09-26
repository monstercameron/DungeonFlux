// Devlog filters: progressive enhancement. Without JS every entry shows.
(function () {
  var nav = document.querySelector('.devlog-filters');
  var entries = document.querySelectorAll('.timeline .entry');
  if (!nav || !entries.length) return;
  nav.hidden = false;

  // Show a count per kind on each chip.
  var counts = { all: entries.length };
  entries.forEach(function (e) { var k = e.dataset.kind; counts[k] = (counts[k] || 0) + 1; });
  nav.querySelectorAll('.chip').forEach(function (chip) {
    var n = counts[chip.dataset.filter] || 0;
    var badge = document.createElement('span');
    badge.className = 'chip-n';
    badge.textContent = String(n);
    chip.append(' ', badge);
  });

  nav.addEventListener('click', function (ev) {
    var chip = ev.target.closest('.chip');
    if (!chip) return;
    var f = chip.dataset.filter;
    nav.querySelectorAll('.chip').forEach(function (c) {
      var on = c === chip;
      c.classList.toggle('is-on', on);
      c.setAttribute('aria-pressed', on ? 'true' : 'false');
    });
    entries.forEach(function (e) { e.hidden = f !== 'all' && e.dataset.kind !== f; });
  });
})();
