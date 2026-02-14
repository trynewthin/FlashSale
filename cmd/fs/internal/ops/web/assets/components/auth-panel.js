// auth-panel 负责访问密钥输入与保存/清空交互。
export function createAuthPanel(container, options) {
  container.innerHTML = `
    <h3>访问密钥</h3>
    <div class="row">
      <input id="ops-access-key" type="password" placeholder="访问密钥（X-Ops-Key）" style="min-width:320px" />
      <button id="ops-save-key">保存密钥</button>
      <button id="ops-clear-key" class="secondary">清空密钥</button>
    </div>
  `;

  const keyInput = container.querySelector("#ops-access-key");
  const saveBtn = container.querySelector("#ops-save-key");
  const clearBtn = container.querySelector("#ops-clear-key");

  keyInput.value = options.initialKey || "";
  saveBtn.addEventListener("click", () => options.onSave(keyInput.value));
  clearBtn.addEventListener("click", () => {
    keyInput.value = "";
    options.onClear();
  });

  return {
    setValue(value) {
      keyInput.value = value || "";
    },
  };
}
