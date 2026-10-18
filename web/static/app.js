(function () {
  "use strict";
  var area = document.getElementById("grid-input");
  if (!area) {
    return;
  }
  area.addEventListener("input", function () {
    var cleaned = area.value.replace(/[^1-9.\r\n]/g, "");
    if (cleaned !== area.value) {
      area.value = cleaned;
    }
  });
  document.querySelectorAll(".sample-btn").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var grid = btn.getAttribute("data-grid") || "";
      if (grid.length !== 81) {
        return;
      }
      area.value = grid.match(/.{9}/g).join("\n");
    });
  });
})();
