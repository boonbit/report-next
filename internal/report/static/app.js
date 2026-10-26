// Minimal web designer: vanilla JS, no bundler.
(function () {
  "use strict";
  var ir = document.getElementById("ir");
  var preview = document.getElementById("preview");

  // Load the embedded hello.frx sample as IR JSON into the editor.
  document.getElementById("btn-sample").addEventListener("click", function () {
    fetch("/api/sample").then(function (res) {
      if (!res.ok) throw new Error("sample failed: " + res.status);
      return res.text();
    }).then(function (text) {
      ir.value = text;
    }).catch(function (err) {
      alert(err.message);
    });
  });

  // Render whatever is in the editor (IR JSON, or pasted .frx XML)
  // and show the resulting HTML document in the preview iframe.
  document.getElementById("btn-render").addEventListener("click", function () {
    fetch("/api/render", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: ir.value
    }).then(function (res) {
      if (!res.ok) throw new Error("render failed: " + res.status);
      return res.text();
    }).then(function (html) {
      // srcdoc keeps the preview self-contained in one iframe.
      preview.srcdoc = html;
    }).catch(function (err) {
      alert(err.message);
    });
  });

  // Export whatever is in the editor as a PDF download.
  document.getElementById("btn-pdf").addEventListener("click", function () {
    fetch("/api/pdf", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: ir.value
    }).then(function (res) {
      if (!res.ok) throw new Error("pdf failed: " + res.status);
      return res.blob();
    }).then(function (blob) {
      var a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = "report.pdf";
      a.click();
      URL.revokeObjectURL(a.href);
    }).catch(function (err) {
      alert(err.message);
    });
  });

  // Upload a local .frx file: import it to IR JSON for editing.
  document.getElementById("file-frx").addEventListener("change", function (ev) {
    var f = ev.target.files && ev.target.files[0];
    if (!f) return;
    f.arrayBuffer().then(function (buf) {
      return fetch("/api/import", { method: "POST", body: buf });
    }).then(function (res) {
      if (!res.ok) throw new Error("import failed: " + res.status);
      return res.text();
    }).then(function (text) {
      ir.value = text;
    }).catch(function (err) {
      alert(err.message);
    });
  });
})();
