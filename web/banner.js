/* SmartGUARD — bandeau de décompte d'expiration (injecté dans les pages de l'application) */
(function () {
  if (window.__smartguard) return;
  window.__smartguard = true;
  var me = document.currentScript || (function () {
    var s = document.getElementsByTagName('script');
    for (var i = s.length - 1; i >= 0; i--) if (/\/banner\.js(\?|$)/.test(s[i].src)) return s[i];
  })();
  if (!me) return;
  var base = me.src.replace(/\/banner\.js(\?.*)?$/, '');
  var KEY = 'smartguard-hidden';

  function get(k) { try { return sessionStorage.getItem(k); } catch (e) { return null; } }
  function set(k, v) { try { sessionStorage.setItem(k, v); } catch (e) {} }
  function esc(s) { return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); }

  var CSS =
    ':host{all:initial}' +
    '.bar{position:fixed;left:0;right:0;top:0;z-index:2147483646;display:flex;align-items:center;gap:14px;' +
    'padding:10px 16px;font:14px/1.4 "Segoe UI",Roboto,Arial,sans-serif;color:#1f1300;background:#ffd66b;' +
    'box-shadow:0 2px 10px rgba(0,0,0,.25);animation:in .35s ease-out}' +
    '.bar.critical{background:#e5484d;color:#fff}.bar.expired{background:#7a0f14;color:#fff}' +
    '@keyframes in{from{transform:translateY(-100%)}to{transform:none}}' +
    '.days{flex:none;min-width:54px;text-align:center;padding:4px 8px;border-radius:8px;background:rgba(0,0,0,.14);font-weight:700;font-size:20px;line-height:1}' +
    '.days small{display:block;font-size:10px;font-weight:600;text-transform:uppercase;letter-spacing:.05em;margin-top:2px}' +
    '.txt{flex:1;min-width:0}.txt b{display:block;font-size:12px;opacity:.85;text-transform:uppercase;letter-spacing:.04em}' +
    '.x{flex:none;border:0;background:rgba(0,0,0,.15);color:inherit;width:30px;height:30px;border-radius:50%;cursor:pointer;font-size:18px;line-height:30px;padding:0}' +
    '.x:hover{background:rgba(0,0,0,.28)}' +
    '.ov{position:fixed;inset:0;z-index:2147483647;background:rgba(20,6,8,.92);display:flex;align-items:center;justify-content:center;padding:16px;font:15px/1.5 "Segoe UI",Roboto,Arial,sans-serif}' +
    '.card{max-width:520px;background:#fff;color:#222;border-radius:14px;padding:28px;text-align:center;box-shadow:0 20px 60px rgba(0,0,0,.5)}' +
    '.card h1{margin:0 0 10px;font-size:22px;color:#7a0f14}.card p{margin:8px 0}.card .c{color:#555;font-size:14px}' +
    '.more{display:block;margin-top:3px;font-size:12px;opacity:.9}' +
    '.rep{flex:none;color:inherit;font-weight:600;font-size:13px;text-decoration:none;border:1px solid currentColor;border-radius:8px;padding:5px 10px;white-space:nowrap}' +
    '.rep:hover{background:rgba(0,0,0,.12)}' +
    '.fab{position:fixed;right:16px;bottom:16px;z-index:2147483645;font:600 13px/1 "Segoe UI",Roboto,Arial,sans-serif;color:#fff;background:#1f5eff;border-radius:99px;padding:10px 14px;text-decoration:none;box-shadow:0 4px 14px rgba(0,0,0,.25)}' +
    '.fab:hover{background:#1748c7}' +
    '.card .tag{display:inline-block;margin:0 0 8px;padding:2px 10px;border-radius:99px;border:1px solid #7a0f14;color:#7a0f14;font-size:13px;font-weight:600}' +
    '.card .rep{display:inline-block;margin-top:12px;color:#fff;background:#7a0f14;border:0}' +
    '@media (max-width:600px){.bar{font-size:13px;padding:8px 10px;gap:10px}.days{font-size:17px;min-width:44px}}';

  function left(d) {
    return d.expired ? 'expiré' : (d.days_left + ' jour' + (d.days_left > 1 ? 's' : ''));
  }
  // Clé de masquage : l'ensemble des échéances affichées et leur décompte.
  function hideKey(st) {
    return (st.deadlines || []).map(function (d) { return d.id + ':' + d.level + ':' + d.days_left; }).join('|') || (st.level + '|' + st.days_left);
  }
  function reportLink(st, cls) {
    return st.report_url ? '<a class="' + cls + '" href="' + esc(st.report_url) + '" target="_blank" rel="noopener">Signaler un problème</a>' : '';
  }

  function render(st) {
    var old = document.getElementById('smartguard-host');
    if (old) old.remove();
    if (!st) return;
    var bar = st.show && (st.blocked || get(KEY) !== hideKey(st));
    if (!bar && !st.report_url) return;

    var host = document.createElement('div');
    host.id = 'smartguard-host';
    var root = host.attachShadow ? host.attachShadow({ mode: 'open' }) : host;
    var title = esc(st.module_name) + (st.software_name ? ' — ' + esc(st.software_name) : '');
    var contact = st.contact ? '<p class="c">Fournisseur : ' + esc(st.contact) + '</p>' : '';
    var html = '<style>' + CSS + '</style>';

    if (st.blocked) {
      html += '<div class="ov" role="alertdialog" aria-modal="true"><div class="card">' +
        '<h1>Accès suspendu</h1>' + (st.blocked_by ? '<div class="tag">Échéance : ' + esc(st.blocked_by) + '</div>' : '') +
        '<p>' + esc(st.message) + '</p>' + contact + reportLink(st, 'rep') + '</div></div>';
    } else if (bar) {
      var d = st.expired ? '!' : st.days_left;
      var unit = st.expired ? 'expiré' : (st.days_left > 1 ? 'jours' : 'jour');
      var others = (st.deadlines || []).filter(function (x) { return x.id !== st.deadline_id; });
      var more = others.length ? '<span class="more">Autre' + (others.length > 1 ? 's' : '') + ' échéance' + (others.length > 1 ? 's' : '') + ' : ' +
        others.map(function (x) { return esc(x.label) + ' — ' + left(x); }).join(' · ') + '</span>' : '';
      var label = (st.deadlines || []).length > 1 && st.label ? ' · ' + esc(st.label) : '';
      html += '<div class="bar ' + esc(st.level) + '" role="alert">' +
        '<div class="days">' + d + '<small>' + unit + '</small></div>' +
        '<div class="txt"><b>' + title + label + '</b>' + esc(st.message) + (st.contact ? ' — ' + esc(st.contact) : '') + more + '</div>' +
        reportLink(st, 'rep') +
        '<button class="x" type="button" title="Masquer pour cette session" aria-label="Fermer">×</button></div>';
    } else {
      html += reportLink(st, 'fab');
    }
    root.innerHTML = html;
    var x = root.querySelector('.x');
    if (x) x.onclick = function () { set(KEY, hideKey(st)); render(Object.assign({}, st, { show: false })); };
    (document.body || document.documentElement).appendChild(host);
  }

  function load() {
    var x = new XMLHttpRequest();
    x.open('GET', base + '/api/status?page=' + encodeURIComponent(location.href), true);
    x.onload = function () { if (x.status === 200) { try { render(JSON.parse(x.responseText)); } catch (e) {} } };
    x.send();
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', load); else load();
  setInterval(load, 10 * 60 * 1000);
})();
