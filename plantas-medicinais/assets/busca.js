(function () {
  // Busca simples no cliente: INDICE = [{t: título, u: url relativa à raiz, s: seção, x: texto}]
  var BASE = (document.querySelector('meta[name="raiz"]') || {}).content || '';
  function norm(s) {
    return (s || '').toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '');
  }
  var docs = (window.INDICE || []).map(function (d) {
    return { t: d.t, u: d.u, s: d.s, x: d.x, nt: norm(d.t), nx: norm(d.x) };
  });
  var input = document.getElementById('q');
  var out = document.getElementById('res');

  function esc(s) {
    return s.replace(/[&<>"]/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
    });
  }
  function trecho(d, termos) {
    var i = -1;
    for (var k = 0; k < termos.length && i < 0; k++) i = d.nx.indexOf(termos[k]);
    if (i < 0) return esc(d.x.slice(0, 180)) + '…';
    var a = Math.max(0, i - 80), b = Math.min(d.x.length, i + 160);
    var t = esc(d.x.slice(a, b));
    termos.forEach(function (w) {
      // destaca ignorando acentos: percorre o texto original comparando versões normalizadas
      var nt = norm(t), j = nt.indexOf(w);
      if (j >= 0 && w.length > 1) t = t.slice(0, j) + '<mark>' + t.slice(j, j + w.length) + '</mark>' + t.slice(j + w.length);
    });
    return (a > 0 ? '…' : '') + t + (b < d.x.length ? '…' : '');
  }
  function buscar() {
    var q = norm(input.value).trim();
    if (q.length < 2) { out.innerHTML = ''; return; }
    var termos = q.split(/\s+/).filter(function (w) { return w.length > 1; });
    var res = [];
    docs.forEach(function (d) {
      var score = 0, ok = true;
      termos.forEach(function (w) {
        var inT = d.nt.indexOf(w) >= 0;
        var n = d.nx.split(w).length - 1;
        if (!inT && !n) ok = false;
        score += (inT ? 20 : 0) + Math.min(n, 10);
      });
      if (ok) res.push({ d: d, score: score });
    });
    res.sort(function (a, b) { return b.score - a.score; });
    if (!res.length) { out.innerHTML = '<p>Nada encontrado para “' + esc(input.value) + '”.</p>'; return; }
    out.innerHTML = '<p class="nres">' + res.length + ' resultado(s)</p>' + res.slice(0, 40).map(function (r) {
      return '<div class="resultado"><a href="' + BASE + r.d.u + '">' + esc(r.d.t) + '</a>' +
        '<div class="sec">' + esc(r.d.s) + '</div><p>' + trecho(r.d, termos) + '</p></div>';
    }).join('');
  }
  input.addEventListener('input', buscar);
  var m = /[?&]q=([^&]*)/.exec(location.search);
  if (m) { input.value = decodeURIComponent(m[1].replace(/\+/g, ' ')); buscar(); }
})();
