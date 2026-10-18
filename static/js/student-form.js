(function () {
  var cls = document.getElementById("class_id");
  var sec = document.getElementById("section_id");
  if (!cls || !sec) {
    return;
  }

  // show only the sections that belong to the chosen class
  function update() {
    var chosen = cls.value;
    Array.prototype.forEach.call(sec.options, function (opt) {
      if (opt.value === "") {
        return;
      }
      var match = opt.getAttribute("data-class") === chosen;
      opt.hidden = !match;
      opt.disabled = !match;
    });
    var current = sec.options[sec.selectedIndex];
    if (current && current.disabled) {
      sec.value = "";
    }
  }

  cls.addEventListener("change", update);
  update();
})();
