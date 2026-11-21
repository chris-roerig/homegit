// Live filter for the repository list. Each word typed must match a repo,
// either as a fuzzy subsequence of its name ("bbot" finds "buenobot") or as a
// substring of its description or topics.
(function () {
  var input = document.getElementById('repo-filter');
  var wrap = document.getElementById('repo-filter-wrap');
  var list = document.getElementById('repo-list');
  if (!input || !wrap || !list) return;

  var items = Array.prototype.slice.call(list.querySelectorAll('[data-repo]'));
  var counter = document.getElementById('repo-count');
  var empty = document.getElementById('repo-empty');

  function subsequence(query, text) {
    var i = 0;
    for (var j = 0; j < text.length && i < query.length; j++) {
      if (text.charAt(j) === query.charAt(i)) i++;
    }
    return i === query.length;
  }

  function matches(words, item) {
    var name = (item.getAttribute('data-name') || '').toLowerCase();
    var text = item.getAttribute('data-text') || '';
    for (var k = 0; k < words.length; k++) {
      if (!subsequence(words[k], name) && text.indexOf(words[k]) === -1) return false;
    }
    return true;
  }

  function update() {
    var words = input.value.toLowerCase().split(/\s+/).filter(Boolean);
    var shown = 0;
    items.forEach(function (item) {
      var ok = matches(words, item);
      item.hidden = !ok;
      if (ok) shown++;
    });
    if (counter) counter.textContent = words.length ? shown + ' of ' + items.length : String(items.length);
    if (empty) empty.hidden = shown !== 0;
  }

  wrap.hidden = false;
  input.addEventListener('input', update);
  input.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') { input.value = ''; update(); }
  });
  update();
})();
