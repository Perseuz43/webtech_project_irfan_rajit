/* ui.js - small usability helpers used on every page.
   1. Sortable table columns      5. (404 page is handled in Go)
   2. Live search above tables    7. Show/hide password buttons
   3. Success messages fade out   9. "Saving..." feedback on form buttons
   4. Active menu link gets aria-current (the highlight itself is in CSS)
*/
(function () {
  "use strict";

  var SEARCH_MIN_ROWS = 5;      // show the search box on tables with at least this many rows
  var FADE_AFTER_MS = 3000;     // success messages disappear after this long
  var RESET_AFTER_MS = 15000;   // re-enable a submit button if nothing happened for this long

  var reduceMotion = window.matchMedia &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---------- helpers ---------- */

  function cellText(cell) {
    var v = cell.getAttribute("data-sort");
    return (v !== null ? v : cell.textContent).replace(/\s+/g, " ").trim();
  }

  function isEmptyValue(s) {
    return s === "" || s === "-";
  }

  // "1,250.50", "$40", "92%" -> number. Anything else -> NaN.
  function toNumber(s) {
    var cleaned = s.replace(/[,\s%$\u09F3\u20AC\u00A3]/g, "");
    return /^-?\d+(\.\d+)?$/.test(cleaned) ? parseFloat(cleaned) : NaN;
  }

  function compareValues(a, b) {
    var na = toNumber(a), nb = toNumber(b);
    if (!isNaN(na) && !isNaN(nb)) {
      return na - nb;
    }
    // numeric:true makes "Class 2" come before "Class 10"
    return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
  }

  function hasFormControls(table) {
    return !!table.querySelector("input:not([type=hidden]), select, textarea");
  }

  /* Work out whether a table can be sorted/searched and return its parts. */
  function inspectTable(table) {
    if (table.classList.contains("profile")) return null;
    if (table.querySelector("table")) return null;               // nested tables are too tricky
    if (table.hasAttribute("data-plain")) return null;           // opt-out switch
    if (hasFormControls(table)) return null;                     // attendance / marks entry

    var rows = Array.prototype.slice.call(table.rows);
    if (rows.length < 3) return null;

    var headerRow = rows[0];
    var headerCells = Array.prototype.slice.call(headerRow.cells);
    if (!headerCells.length || !headerCells.every(function (c) { return c.tagName === "TH"; })) {
      return null;
    }

    var bodyRows = rows.slice(1);
    var ok = bodyRows.every(function (r) {
      return r.cells.length === headerCells.length &&
        !r.querySelector("th") &&
        !Array.prototype.some.call(r.cells, function (c) { return c.colSpan > 1; });
    });
    if (!ok) return null;

    return { headerRow: headerRow, headerCells: headerCells, bodyRows: bodyRows };
  }

  /* ---------- 1. sortable columns ---------- */

  var SKIP_HEADERS = /^(actions?|manage|edit|delete|receipt|view)$/i;

  function makeSortable(table, info) {
    var bodyParent = info.bodyRows[0].parentNode;
    info.bodyRows.forEach(function (row, i) { row.setAttribute("data-idx", i); });

    var current = { col: -1, dir: 0 };   // dir: 1 = A-Z, -1 = Z-A, 0 = original order

    info.headerCells.forEach(function (th, col) {
      var label = th.textContent.trim();
      if (!label || SKIP_HEADERS.test(label)) return;
      var first = info.bodyRows[0].cells[col];
      if (first && first.querySelector("form, button")) return;

      th.classList.add("sortable");
      th.setAttribute("tabindex", "0");
      th.setAttribute("role", "button");
      th.setAttribute("title", "Click to sort");

      function activate() {
        var dir = current.col === col ? (current.dir === 1 ? -1 : current.dir === -1 ? 0 : 1) : 1;
        current = { col: col, dir: dir };
        sortBy(col, dir);
        info.headerCells.forEach(function (h) {
          h.removeAttribute("aria-sort");
          h.classList.remove("sorted-asc", "sorted-desc");
        });
        if (dir === 1) { th.setAttribute("aria-sort", "ascending"); th.classList.add("sorted-asc"); }
        if (dir === -1) { th.setAttribute("aria-sort", "descending"); th.classList.add("sorted-desc"); }
      }

      th.addEventListener("click", activate);
      th.addEventListener("keydown", function (e) {
        if (e.key === "Enter" || e.key === " ") { e.preventDefault(); activate(); }
      });
    });

    function sortBy(col, dir) {
      var rows = info.bodyRows.slice();
      if (dir === 0) {
        rows.sort(function (a, b) { return a.getAttribute("data-idx") - b.getAttribute("data-idx"); });
      } else {
        rows.sort(function (a, b) {
          var va = cellText(a.cells[col]), vb = cellText(b.cells[col]);
          var ea = isEmptyValue(va), eb = isEmptyValue(vb);
          if (ea || eb) return ea === eb ? 0 : ea ? 1 : -1;      // blanks always last
          return dir * compareValues(va, vb);
        });
      }
      rows.forEach(function (r) { bodyParent.appendChild(r); });
    }
  }

  /* ---------- 2. live search ---------- */

  function addSearch(table, info) {
    if (info.bodyRows.length < SEARCH_MIN_ROWS) return;

    var tools = document.createElement("div");
    tools.className = "table-tools";

    var input = document.createElement("input");
    input.type = "search";
    input.placeholder = "Search this list...";
    input.setAttribute("aria-label", "Search this list");
    input.autocomplete = "off";

    var count = document.createElement("span");
    count.className = "table-count muted small";
    count.setAttribute("aria-live", "polite");

    var none = document.createElement("p");
    none.className = "muted table-empty";
    none.hidden = true;

    tools.appendChild(input);
    tools.appendChild(count);
    table.parentNode.insertBefore(tools, table);
    table.parentNode.insertBefore(none, table.nextSibling);

    info.bodyRows.forEach(function (r) {
      r._text = r.textContent.replace(/\s+/g, " ").toLowerCase();
    });

    function filter() {
      var q = input.value.trim().toLowerCase();
      var shown = 0;
      info.bodyRows.forEach(function (r) {
        var match = q === "" || r._text.indexOf(q) !== -1;
        r.hidden = !match;
        if (match) shown++;
      });
      count.textContent = q === "" ? "" : shown + " of " + info.bodyRows.length;
      none.hidden = shown !== 0;
      none.textContent = shown === 0 ? 'Nothing matches "' + input.value.trim() + '".' : "";
      table.hidden = shown === 0;
    }

    input.addEventListener("input", filter);
    input.addEventListener("keydown", function (e) {
      if (e.key === "Escape") { input.value = ""; filter(); }
    });
  }

  function setupTables() {
    var tables = document.querySelectorAll("main table");
    Array.prototype.forEach.call(tables, function (table) {
      var info = inspectTable(table);
      if (!info) return;
      makeSortable(table, info);
      addSearch(table, info);
    });
  }

  /* ---------- 3. success messages fade out ---------- */

  function setupFade() {
    var messages = document.querySelectorAll(".success:not([data-keep])");
    Array.prototype.forEach.call(messages, function (el) {
      setTimeout(function () {
        if (reduceMotion) { el.remove(); return; }
        el.style.maxHeight = el.offsetHeight + "px";
        void el.offsetHeight;                       // make the browser register the start height
        el.classList.add("fading");
        setTimeout(function () { el.remove(); }, 600);
      }, FADE_AFTER_MS);
    });
  }

  /* ---------- 4. menu: tell screen readers which page is current ---------- */

  function setupNav() {
    var active = document.querySelector(".nav a.active");
    if (active) active.setAttribute("aria-current", "page");
  }

  /* ---------- 7. show / hide password ---------- */

  function setupPasswordToggles() {
    var fields = document.querySelectorAll('input[type="password"]');
    Array.prototype.forEach.call(fields, function (input) {
      if (input.parentNode.classList.contains("pw-wrap")) return;

      var wrap = document.createElement("div");
      wrap.className = "pw-wrap";
      input.parentNode.insertBefore(wrap, input);
      wrap.appendChild(input);

      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "pw-toggle";
      btn.textContent = "Show";
      btn.setAttribute("aria-label", "Show password");
      btn.setAttribute("aria-pressed", "false");
      wrap.appendChild(btn);

      btn.addEventListener("click", function () {
        var show = input.type === "password";
        input.type = show ? "text" : "password";
        btn.textContent = show ? "Hide" : "Show";
        btn.setAttribute("aria-label", show ? "Hide password" : "Show password");
        btn.setAttribute("aria-pressed", show ? "true" : "false");
      });
    });
  }

  /* ---------- 9. "Saving..." feedback ---------- */

  function loadingText(form, btn) {
    var custom = btn.getAttribute("data-loading");
    if (custom) return custom;
    var action = (form.getAttribute("action") || "");
    if (action === "/login") return "Signing in...";
    if (action === "/logout") return "Logging out...";
    if (/\/delete$/.test(action) || btn.classList.contains("danger")) return "Deleting...";
    return "Saving...";
  }

  function resetButton(btn) {
    if (!btn || btn.getAttribute("data-busy") !== "1") return;
    btn.disabled = false;
    btn.classList.remove("is-loading");
    btn.textContent = btn.getAttribute("data-label");
    btn.removeAttribute("data-busy");
    if (btn.form) btn.form.removeAttribute("data-submitting");
  }

  function setupSubmitFeedback() {
    document.addEventListener("submit", function (e) {
      var form = e.target;
      if (!(form instanceof HTMLFormElement)) return;
      if (form.getAttribute("data-submitting") === "1") { e.preventDefault(); return; }   // double click
      if (e.defaultPrevented) return;                                  // e.g. the user pressed Cancel in confirm()
      if ((form.method || "").toLowerCase() !== "post") return;       // search/filter forms are fast
      if (form.target === "_blank") return;

      var btn = e.submitter || form.querySelector('button[type="submit"], button:not([type])');
      if (!btn) return;

      form.setAttribute("data-submitting", "1");

      // Wait one tick: the browser must read the form values (and the clicked
      // button's own name/value) BEFORE we disable the button.
      setTimeout(function () {
        btn.setAttribute("data-label", btn.textContent);
        btn.setAttribute("data-busy", "1");
        btn.textContent = loadingText(form, btn);
        btn.classList.add("is-loading");
        btn.disabled = true;
        setTimeout(function () { resetButton(btn); }, RESET_AFTER_MS);
      }, 0);
    });

    // Back button: the browser may bring the page back with buttons still disabled.
    window.addEventListener("pageshow", function (e) {
      if (!e.persisted) return;
      Array.prototype.forEach.call(document.querySelectorAll('button[data-busy="1"]'), resetButton);
    });
  }

  /* ---------- go ---------- */

  setupNav();
  setupTables();
  setupFade();
  setupPasswordToggles();
  setupSubmitFeedback();
})();
