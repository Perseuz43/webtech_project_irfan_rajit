(function () {
  var root = document.documentElement;
  var btn = document.getElementById("theme-toggle");
  if (!btn) {
    return;
  }

  // the theme currently on screen, even when it comes from the system setting
  function effective() {
    var t = root.getAttribute("data-theme");
    if (t === "light" || t === "dark") {
      return t;
    }
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  btn.addEventListener("click", function () {
    var next = effective() === "dark" ? "light" : "dark";
    root.setAttribute("data-theme", next); // switch instantly

    // save it on the server; the server stores it on the user's account when logged in
    fetch("/theme", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: "theme=" + next,
      credentials: "same-origin"
    }).catch(function () {});
  });
})();
