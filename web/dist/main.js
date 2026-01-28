const uploadZone = document.getElementById("upload-zone");
const selectButton = document.getElementById("select-button");
const fileInput = document.getElementById("file-input");
const uploadHint = document.getElementById("upload-hint");
const taskIdEl = document.getElementById("task-id");
const statusEl = document.getElementById("task-status");
const progressEl = document.getElementById("task-progress");
const statusMessage = document.getElementById("status-message");
const resultLink = document.getElementById("result-link");
const queryInput = document.getElementById("query-input");
const queryButton = document.getElementById("query-button");
const queryMessage = document.getElementById("query-message");
let pollTimer;
const setMessage = (el, message, isError = false) => {
  el.textContent = message;
  el.classList.toggle("error", isError);
};
const setCurrentTask = (taskId) => {
  taskIdEl.textContent = taskId;
  statusEl.textContent = "处理中";
  progressEl.textContent = "-";
  resultLink.classList.add("hidden");
  resultLink.href = "#";
  setMessage(statusMessage, "");
};
const startPolling = (taskId) => {
  if (pollTimer) {
    window.clearInterval(pollTimer);
  }
  pollTimer = window.setInterval(() => {
    void fetchStatus(taskId);
  }, 2000);
};
const normalizeStatus = (status) => status.toLowerCase();
const fetchStatus = async (taskId) => {
  try {
    const res = await fetch(`/api/tasks/${encodeURIComponent(taskId)}`);
    if (!res.ok) {
      throw new Error(`状态查询失败: ${res.status}`);
    }
    const data = await res.json();
    const status = normalizeStatus(data.status ?? data.state ?? "unknown");
    const progress = data.completed_count ?? data.progress ?? "";
    statusEl.textContent = status;
    progressEl.textContent = progress || "-";
    if (status === "completed" || status === "success" || status === "done") {
      resultLink.href = `/api/tasks/${encodeURIComponent(taskId)}/result`;
      resultLink.classList.remove("hidden");
      setMessage(statusMessage, "任务完成，可以下载结果。");
      if (pollTimer) {
        window.clearInterval(pollTimer);
        pollTimer = undefined;
      }
    } else if (status === "failed") {
      setMessage(statusMessage, data.error ?? "任务失败。", true);
      if (pollTimer) {
        window.clearInterval(pollTimer);
        pollTimer = undefined;
      }
    } else {
      setMessage(statusMessage, "处理中，请稍候...");
    }
    return status;
  } catch (err) {
    const message = err instanceof Error ? err.message : "状态查询失败";
    setMessage(statusMessage, message, true);
    return null;
  }
};
const uploadFile = async (file) => {
  setMessage(uploadHint, `正在上传：${file.name}`);
  try {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch("/api/tasks", {
      method: "POST",
      body: formData,
    });
    if (!res.ok) {
      throw new Error(`上传失败: ${res.status}`);
    }
    const data = await res.json();
    const taskId = data.task_id ?? data.id ?? data.taskId;
    if (!taskId) {
      throw new Error("返回中未找到 task_id");
    }
    setCurrentTask(taskId);
    startPolling(taskId);
    setMessage(uploadHint, "上传成功，开始处理。");
  } catch (err) {
    const message = err instanceof Error ? err.message : "上传失败";
    setMessage(uploadHint, message, true);
  }
};
const handleFiles = (files) => {
  if (!files || files.length === 0) {
    return;
  }
  const file = files[0];
  if (file.type !== "application/pdf") {
    setMessage(uploadHint, "请选择 PDF 文件。", true);
    return;
  }
  void uploadFile(file);
};
uploadZone.addEventListener("dragover", (event) => {
  event.preventDefault();
  uploadZone.classList.add("dragover");
});
uploadZone.addEventListener("dragleave", () => {
  uploadZone.classList.remove("dragover");
});
uploadZone.addEventListener("drop", (event) => {
  event.preventDefault();
  uploadZone.classList.remove("dragover");
  handleFiles(event.dataTransfer?.files ?? null);
});
selectButton.addEventListener("click", () => {
  fileInput.click();
});
fileInput.addEventListener("change", () => {
  handleFiles(fileInput.files);
});
queryButton.addEventListener("click", async () => {
  const taskId = queryInput.value.trim();
  if (!taskId) {
    setMessage(queryMessage, "请输入 Task ID。", true);
    return;
  }
  setMessage(queryMessage, "");
  setCurrentTask(taskId);
  const status = await fetchStatus(taskId);
  if (status && status !== "completed" && status !== "success" && status !== "done" && status !== "failed") {
    startPolling(taskId);
  }
});
