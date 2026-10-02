/*
 * Reports Keycloak error pages to Sentry / GlitchTip.
 *
 * A reader who lands on a Keycloak error page («Invalid parameter:
 * redirect_uri», expired links …) is invisible to the reader app, so the page
 * reports itself.  No SDK: one event per page view through the envelope
 * endpoint, sent as text/plain so the browser needs no CORS preflight.
 *
 * Sent: the error text, the page, client_id, redirect_uri (path only, no
 * query), response_mode/prompt and the user agent.  Never sent: codes, state,
 * session/tab IDs, form input, cookies.
 *
 * Plain ES5 on purpose – it has to run on old iPhones too.
 */
(function () {
  'use strict';

  function meta(name) {
    var el = document.querySelector('meta[name="' + name + '"]');
    return el ? (el.getAttribute('content') || '').trim() : '';
  }

  function parseDsn(dsn) {
    var m = /^(https?):\/\/([^@\/]+)@([^\/]+)\/(?:(.*)\/)?(\d+)$/.exec(dsn);
    if (!m) return null;
    var path = m[4] ? '/' + m[4] : '';
    return {
      url: m[1] + '://' + m[3] + path + '/api/' + m[5] + '/envelope/?sentry_key=' +
           encodeURIComponent(m[2]) + '&sentry_version=7',
      dsn: dsn
    };
  }

  function uuid() {
    var s = '';
    for (var i = 0; i < 32; i++) s += Math.floor(Math.random() * 16).toString(16);
    return s;
  }

  function stripQuery(u) {
    return (u || '').split('#')[0].split('?')[0];
  }

  /* Only harmless OAuth parameters survive; everything else is dropped. */
  function safeParams() {
    var keep = { client_id: 1, redirect_uri: 1, response_mode: 1, response_type: 1, prompt: 1, scope: 1, ui_locales: 1 };
    var out = {};
    var q = window.location.search.replace(/^\?/, '').split('&');
    for (var i = 0; i < q.length; i++) {
      if (!q[i]) continue;
      var kv = q[i].split('=');
      var k = decodeURIComponent(kv[0] || '');
      if (!keep[k]) continue;
      var v = decodeURIComponent((kv[1] || '').replace(/\+/g, ' '));
      out[k] = k === 'redirect_uri' ? stripQuery(v) : v.slice(0, 200);
    }
    return out;
  }

  function text(sel) {
    var el = document.querySelector(sel);
    return el ? (el.textContent || '').replace(/\s+/g, ' ').trim() : '';
  }

  function problem() {
    var page = (document.body && document.body.getAttribute('data-page-id')) || '';
    if (page === 'login-error' || page === 'login-webauthn-error') {
      return { page: page, level: 'error',
               text: text('#kc-error-message') || text('#kc-page-title') || 'Fehlerseite' };
    }
    if (page === 'login-login-page-expired') {
      return { page: page, level: 'warning', text: text('#kc-page-title') || 'Seite abgelaufen' };
    }
    /* Server-side failures shown inside a normal page (not wrong passwords). */
    var alert = text('.alert-error .kc-feedback-text');
    if (alert && /redirect|client|unexpected|interner|internal|abgelaufen|expired|cookie/i.test(alert)) {
      return { page: page, level: 'warning', text: alert };
    }
    return null;
  }

  function send(target, envelope) {
    try {
      if (navigator.sendBeacon && navigator.sendBeacon(target.url, envelope)) return;
    } catch (e) { /* fall through */ }
    try {
      if (window.fetch) {
        window.fetch(target.url, { method: 'POST', body: envelope, mode: 'no-cors', keepalive: true,
                                   headers: { 'Content-Type': 'text/plain;charset=UTF-8' } });
        return;
      }
      var xhr = new XMLHttpRequest();
      xhr.open('POST', target.url, true);
      xhr.setRequestHeader('Content-Type', 'text/plain;charset=UTF-8');
      xhr.send(envelope);
    } catch (e) { /* reporting must never break the login page */ }
  }

  function report() {
    var target = parseDsn(meta('augustin-sentry-dsn'));
    if (!target) return;
    var p = problem();
    if (!p) return;

    var params = safeParams();
    var id = uuid();
    var now = new Date().toISOString();
    var event = {
      event_id: id,
      timestamp: now,
      platform: 'javascript',
      level: p.level,
      logger: 'keycloak-theme',
      environment: meta('augustin-sentry-env') || 'production',
      message: { formatted: 'Keycloak: ' + p.text },
      fingerprint: ['keycloak', p.page, p.text],
      tags: {
        component: 'keycloak',
        page: p.page,
        client_id: params.client_id || '',
        locale: document.documentElement.lang || ''
      },
      extra: {
        oauth_params: params,
        referrer: stripQuery(document.referrer)
      },
      request: {
        url: window.location.origin + window.location.pathname,
        headers: { 'User-Agent': navigator.userAgent }
      }
    };
    var envelope =
      JSON.stringify({ event_id: id, sent_at: now, dsn: target.dsn }) + '\n' +
      JSON.stringify({ type: 'event' }) + '\n' +
      JSON.stringify(event);
    send(target, envelope);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', report);
  } else {
    report();
  }
})();
