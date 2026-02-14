// status-bar 统一展示页面级成功/失败提示。
export function createStatusBar(container) {
  container.innerHTML = "";
  const box = document.createElement("div");
  box.style.display = "none";
  container.appendChild(box);

  function show(kind, message) {
    box.style.display = "block";
    box.className = kind === "error" ? "status status-error" : "status status-ok";
    box.textContent = message || "";
  }

  function clear() {
    box.style.display = "none";
    box.textContent = "";
  }

  return {
    showOK(message) {
      show("ok", message);
    },
    showError(message) {
      show("error", message);
    },
    clear,
  };
}
