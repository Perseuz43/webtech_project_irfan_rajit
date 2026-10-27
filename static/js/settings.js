(function () {
  var color = document.getElementById("accent");
  var useDefault = document.getElementById("use_default");
  var preview = document.getElementById("preview-button");
  if (!color || !useDefault || !preview) {
    return;
  }

  function update() {
    preview.style.background = useDefault.checked ? "#2563eb" : color.value;
  }

  var swatches = document.querySelectorAll(".swatch");
  for (var i = 0; i < swatches.length; i++) {
    (function (button) {
      button.style.background = button.getAttribute("data-color");
      button.addEventListener("click", function () {
        color.value = button.getAttribute("data-color");
        useDefault.checked = false;
        update();
      });
    })(swatches[i]);
  }

  color.addEventListener("input", function () {
    useDefault.checked = false;
    update();
  });
  useDefault.addEventListener("change", update);
  update();
})();
