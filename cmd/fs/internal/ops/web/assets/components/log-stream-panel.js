// log-stream-panel 管理日志快照显示与 SSE 实时订阅。
export function createLogStreamPanel(container) {
  container.innerHTML = `
    <div class="row" style="margin-bottom:8px">
      <button id="ops-log-stop" class="secondary">停止流</button>
    </div>
    <textarea id="ops-log-textarea" readonly></textarea>
  `;

  const textarea = container.querySelector("#ops-log-textarea");
  const stopBtn = container.querySelector("#ops-log-stop");

  let eventSource = null;
  stopBtn.addEventListener("click", () => stopStream());

  function setText(text) {
    textarea.value = text || "";
    textarea.scrollTop = textarea.scrollHeight;
  }

  function appendLine(line) {
    textarea.value += `${line}\n`;
    textarea.scrollTop = textarea.scrollHeight;
  }

  function stopStream() {
    if (eventSource) {
      try {
        eventSource.close();
      } catch (_err) {
        // ignore
      }
      eventSource = null;
    }
  }

  function startStream(url, handlers = {}) {
    stopStream();
    eventSource = new EventSource(url);
    eventSource.addEventListener("log", (event) => {
      try {
        const payload = JSON.parse(event.data || "{}");
        if (payload.line) {
          appendLine(payload.line);
        }
      } catch (_err) {
        // ignore malformed frames
      }
    });
    eventSource.addEventListener("ping", () => {});
    eventSource.onerror = () => {
      if (handlers.onError) {
        handlers.onError();
      }
    };
  }

  return {
    setText,
    appendLine,
    startStream,
    stopStream,
  };
}
