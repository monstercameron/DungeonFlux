// DungeonFlux product page. Without JS every section still reads;
// the narration type-in, the pinned TV and the roll are enhancements.

// Where "Start a table" goes. Point this at the hosted table when one exists.
var TRY_URL = 'https://play.dungeonfluxdnd.com/dm';

(function () {
  var reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  document.querySelectorAll('.js-try').forEach(function (a) { a.href = TRY_URL; });

  // Top bar turns solid once the hero is behind you.
  var bar = document.getElementById('bar');
  function onScroll() { bar.classList.toggle('is-solid', window.scrollY > 40); }
  onScroll();
  window.addEventListener('scroll', onScroll, { passive: true });

  // The page's one orchestrated moment: the dungeon master speaks the opening line.
  var narration = document.getElementById('narration');
  var panel = narration && narration.closest('.narration');
  var DM_TEXT = narration ? narration.textContent : '';
  if (narration && !reduced) {
    var full = narration.textContent;
    narration.style.minHeight = narration.offsetHeight + 'px';
    narration.textContent = '';
    var caret = document.createElement('span');
    caret.className = 'caret';
    var shown = document.createTextNode('');
    narration.append(shown, caret);
    var i = 0;
    setTimeout(function step() {
      i += 1;
      shown.data = full.slice(0, i);
      if (i < full.length) { setTimeout(step, full[i - 1] === ',' || full[i - 1] === '.' ? 180 : 26); }
      else { caret.remove(); panel.classList.add('is-done'); setTimeout(heroLoop, 1800); }
    }, 700);
  } else if (panel) {
    panel.classList.add('is-done');
  }

  // Then the pitch on a loop: the phone taps, the TV answers, the dungeon master resumes.
  var who = document.getElementById('narr-who');
  var avatar = document.getElementById('narr-avatar');
  var body = panel && panel.querySelector('.narration-body');
  var talk = document.getElementById('pocket-talk');
  var DM = { who: 'Dungeon Master', img: 'assets/dm-avatar.webp', text: DM_TEXT };
  var VELL = { who: 'Mother Vell', img: 'assets/vell-avatar.webp', text: '“Lamplighter? Never heard of him.” She keeps wiping the same spot on the bar.' };
  new Image().src = VELL.img;

  function speaker(v) {
    body.classList.add('is-swapping');
    avatar.classList.add('is-swapping');
    setTimeout(function () {
      who.textContent = v.who;
      avatar.src = v.img;
      narration.textContent = v.text;
      panel.classList.remove('is-done');
      body.classList.remove('is-swapping');
      avatar.classList.remove('is-swapping');
      setTimeout(function () { panel.classList.add('is-done'); }, 2400);
    }, 300);
  }

  function heroLoop() {
    if (!talk || !body) return;
    talk.classList.add('is-pressed');
    setTimeout(function () { talk.classList.remove('is-pressed'); speaker(VELL); }, 350);
    setTimeout(function () { speaker(DM); }, 6500);
    setTimeout(heroLoop, 13000);
  }

  // One night at the table: the pinned TV shows whichever beat is in view.
  var shots = document.querySelectorAll('.tv-shot, .mini-screen');
  var beats = document.querySelectorAll('.beat');
  if ('IntersectionObserver' in window && shots.length) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (!e.isIntersecting) return;
        var n = e.target.dataset.shot;
        beats.forEach(function (b) { b.classList.toggle('is-on', b === e.target); });
        shots.forEach(function (s) { s.classList.toggle('is-on', s.dataset.shot === n); });
      });
    }, { rootMargin: '-45% 0px -45% 0px' });
    beats.forEach(function (b) { io.observe(b); });
  }

  // The reel: gilt arrows instead of a browser scrollbar.
  var reel = document.getElementById('reel');
  var reelBtns = document.querySelectorAll('.reel-btn');
  function reelState() {
    var max = reel.scrollWidth - reel.clientWidth - 2;
    reelBtns[0].disabled = reel.scrollLeft <= 2;
    reelBtns[1].disabled = reel.scrollLeft >= max;
  }
  if (reel && reelBtns.length === 2) {
    reelBtns.forEach(function (b) {
      b.addEventListener('click', function () {
        var card = reel.querySelector('li');
        var step = card ? card.getBoundingClientRect().width + 20 : reel.clientWidth * 0.8;
        reel.scrollBy({ left: step * Number(b.dataset.dir), behavior: reduced ? 'auto' : 'smooth' });
        setTimeout(reelState, reduced ? 0 : 500);
      });
    });
    reel.addEventListener('scroll', reelState, { passive: true });
    window.addEventListener('resize', reelState);
    reelState();
  }

  // The roll: the phone acts, the TV shows the die, the story follows the result.
  var btn = document.getElementById('roll-btn');
  if (!btn) return;
  var again = document.getElementById('again');
  var after = document.getElementById('after');
  var scene = document.querySelector('.scene');
  var num = document.getElementById('d20-num');
  var verdict = document.getElementById('die-verdict');
  var math = document.getElementById('check-math');
  var subtitle = document.getElementById('subtitle');
  var subWho = document.getElementById('subtitle-who');
  var subText = document.getElementById('subtitle-text');
  var BONUS = 3, DC = 12;
  var OPENING = subText.textContent;
  var START_MATH = math.textContent;

  // The same beat as the live demo: success reveals the secret; failure reroutes the story.
  var WIN = '“They took him up the bell tower.” She glances at the door. “You didn’t hear it from me.”';
  var LOSE = '“I pour drinks. I don’t answer questions.” Behind you, the door bangs open. A stranger holds up a letter with your name on it.';

  function d20() {
    var a = new Uint32Array(1);
    window.crypto.getRandomValues(a);
    return (a[0] % 20) + 1;
  }

  function say(who, text) {
    subWho.textContent = who;
    subText.textContent = text;
    subtitle.classList.remove('is-new');
    void subtitle.offsetWidth;
    subtitle.classList.add('is-new');
  }

  var byKey = false;
  btn.addEventListener('click', function (e) {
    byKey = e.detail === 0;
    var n = d20();
    var total = n + BONUS;
    var hit = total >= DC;
    btn.disabled = true;
    scene.classList.remove('is-hit', 'is-miss', 'is-settled');
    verdict.replaceChildren();
    math.textContent = 'Rolling on the TV…';
    void scene.offsetWidth;
    scene.classList.add('is-rolling');
    var spin = reduced ? 0 : setInterval(function () { num.textContent = String(d20()); }, 60);
    setTimeout(function () {
      clearInterval(spin);
      num.textContent = String(n);
      scene.classList.remove('is-rolling');
      scene.classList.add('is-settled', hit ? 'is-hit' : 'is-miss');
      var label = document.createElement('b');
      label.textContent = hit ? 'Success' : 'Failure';
      verdict.replaceChildren(n + ' + 3 = ' + total, label);
      math.textContent = 'You rolled ' + n + ', plus 3 is ' + total + '. ' + (hit ? 'That beats 12.' : 'That falls short of 12.');
      setTimeout(function () {
        say(hit ? 'Mother Vell' : 'Mother Vell, then a stranger', hit ? WIN : LOSE);
        btn.hidden = true;
        after.hidden = false;
        if (byKey) again.focus({ preventScroll: true });
      }, reduced ? 0 : 450);
    }, reduced ? 0 : 750);
  });

  again.addEventListener('click', function (e) {
    scene.classList.remove('is-hit', 'is-miss', 'is-settled', 'is-rolling');
    verdict.replaceChildren();
    math.textContent = START_MATH;
    say('Mother Vell', OPENING);
    after.hidden = true;
    btn.hidden = false;
    btn.disabled = false;
    if (e.detail === 0) btn.focus({ preventScroll: true });
  });
})();
