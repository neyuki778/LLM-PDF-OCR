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
const authLink = document.getElementById("auth-link");
const authUser = document.getElementById("auth-user");
const authUserEmail = document.getElementById("auth-user-email");
const logoutButton = document.getElementById("logout-button");
const authBenefitText = document.getElementById("auth-benefit-text");
const historyCard = document.getElementById("history-card");
const historyHint = document.getElementById("history-hint");
const historyList = document.getElementById("history-list");
const historyRefreshButton = document.getElementById("history-refresh-button");
let pollTimer;
let refreshPromise = null;
let currentUser = null;
const setMessage = (el, message, isError = false) => {
    el.textContent = message;
    el.classList.toggle("error", isError);
};
const setAuthBenefit = (user) => {
    if (user) {
        authBenefitText.textContent = "已登录：历史任务会自动保存。登录用户单次额度更高（最多 40 页），并可使用额外功能。";
        return;
    }
    authBenefitText.textContent = "游客模式不保存历史任务记录。登录 / 注册后可获得更高额度（单次最多 40 页）并使用历史任务等额外功能。";
};
const clearHistoryList = () => {
    historyList.innerHTML = "";
};
const formatStatus = (status) => {
    const normalized = status.toLowerCase();
    if (normalized === "completed")
        return "已完成";
    if (normalized === "processing")
        return "处理中";
    if (normalized === "pending")
        return "排队中";
    if (normalized === "failed")
        return "失败";
    return status;
};
const formatUnixTime = (sec) => {
    if (!Number.isFinite(sec) || sec <= 0) {
        return "-";
    }
    return new Date(sec * 1000).toLocaleString("zh-CN", { hour12: false });
};
const normalizeStatus = (status) => status.toLowerCase();
const runTaskQuery = async (taskId) => {
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
};
const renderHistoryList = (items) => {
    clearHistoryList();
    for (const item of items) {
        const li = document.createElement("li");
        li.className = "history-item";
        const left = document.createElement("div");
        left.className = "history-main";
        const id = document.createElement("span");
        id.className = "history-id";
        id.textContent = item.task_id;
        const meta = document.createElement("span");
        meta.className = "history-meta";
        meta.textContent = `${formatStatus(item.status)} · ${formatUnixTime(item.created_at)}`;
        left.append(id, meta);
        const btn = document.createElement("button");
        btn.type = "button";
        btn.textContent = "查看";
        btn.addEventListener("click", () => {
            queryInput.value = item.task_id;
            setMessage(queryMessage, "已从历史任务载入。");
            void runTaskQuery(item.task_id);
        });
        li.append(left, btn);
        historyList.appendChild(li);
    }
};
const loadTaskHistory = async () => {
    if (!currentUser) {
        clearHistoryList();
        setMessage(historyHint, "登录后可查看历史任务。");
        return;
    }
    setMessage(historyHint, "正在加载历史任务...");
    try {
        const res = await fetchWithAutoRefresh("/api/tasks/history?limit=20");
        if (!res.ok) {
            if (res.status === 401) {
                setCurrentUser(null);
                return;
            }
            throw new Error(`历史任务加载失败: ${res.status}`);
        }
        const data = (await res.json());
        const items = Array.isArray(data.items) ? data.items : [];
        if (items.length === 0) {
            clearHistoryList();
            setMessage(historyHint, "暂无历史任务。");
            return;
        }
        renderHistoryList(items);
        setMessage(historyHint, `已加载最近 ${items.length} 条历史任务。`);
    }
    catch (err) {
        const message = err instanceof Error ? err.message : "历史任务加载失败";
        clearHistoryList();
        setMessage(historyHint, message, true);
    }
};
const setCurrentUser = (user) => {
    currentUser = user;
    setAuthBenefit(user);
    if (user) {
        authUserEmail.textContent = user.email;
        authLink.classList.add("hidden");
        authUser.classList.remove("hidden");
        historyCard.classList.remove("hidden");
        void loadTaskHistory();
        return;
    }
    authUserEmail.textContent = "";
    authUser.classList.add("hidden");
    authLink.classList.remove("hidden");
    historyCard.classList.add("hidden");
    clearHistoryList();
    setMessage(historyHint, "登录后可查看历史任务。");
};
const refreshSessionSilently = async () => {
    if (refreshPromise) {
        return refreshPromise;
    }
    refreshPromise = (async () => {
        try {
            const res = await fetch("/api/auth/refresh", {
                method: "POST",
                credentials: "include",
            });
            if (!res.ok) {
                setCurrentUser(null);
                return false;
            }
            return true;
        }
        catch (_) {
            setCurrentUser(null);
            return false;
        }
    })().finally(() => {
        refreshPromise = null;
    });
    return refreshPromise;
};
const fetchWithAutoRefresh = async (url, init = {}, canRetry = true) => {
    const res = await fetch(url, {
        ...init,
        credentials: "include",
    });
    if (res.status === 401 && canRetry && url !== "/api/auth/refresh") {
        const refreshed = await refreshSessionSilently();
        if (refreshed) {
            return fetchWithAutoRefresh(url, init, false);
        }
    }
    return res;
};
const loadCurrentUser = async () => {
    try {
        const res = await fetchWithAutoRefresh("/api/auth/me");
        if (!res.ok) {
            setCurrentUser(null);
            return;
        }
        const data = await res.json();
        const user = data === null || data === void 0 ? void 0 : data.user;
        if (!user || typeof user.email !== "string") {
            setCurrentUser(null);
            return;
        }
        setCurrentUser(user);
    }
    catch (_) {
        setCurrentUser(null);
    }
};
const handleLogout = async () => {
    try {
        await fetch("/api/auth/logout", {
            method: "POST",
            credentials: "include",
        });
    }
    catch (_) {
        // ignore network error
    }
    setCurrentUser(null);
};
const setCurrentTask = (taskId) => {
    taskIdEl.textContent = taskId;
    statusEl.textContent = "处理中";
    progressEl.textContent = "-";
    resultLink.classList.add("hidden");
    resultLink.href = "#";
    resultLink.removeAttribute("download");
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
const fetchStatus = async (taskId) => {
    var _a, _b, _c, _d, _e;
    try {
        const res = await fetchWithAutoRefresh(`/api/tasks/${encodeURIComponent(taskId)}`);
        if (!res.ok) {
            if (res.status === 401) {
                throw new Error("登录已过期，请重新登录。");
            }
            throw new Error(`状态查询失败: ${res.status}`);
        }
        const data = await res.json();
        const status = normalizeStatus((_b = (_a = data.status) !== null && _a !== void 0 ? _a : data.state) !== null && _b !== void 0 ? _b : "unknown");
        const progress = (_d = (_c = data.completed_count) !== null && _c !== void 0 ? _c : data.progress) !== null && _d !== void 0 ? _d : "";
        statusEl.textContent = status;
        progressEl.textContent = progress || "-";
        if (status === "completed" || status === "success" || status === "done") {
            resultLink.href = `/api/tasks/${encodeURIComponent(taskId)}/result`;
            resultLink.setAttribute("download", `${taskId}.md`);
            resultLink.classList.remove("hidden");
            setMessage(statusMessage, "任务完成，可以下载结果。");
            if (pollTimer) {
                window.clearInterval(pollTimer);
                pollTimer = undefined;
            }
        }
        else if (status === "failed") {
            setMessage(statusMessage, (_e = data.error) !== null && _e !== void 0 ? _e : "任务失败。", true);
            if (pollTimer) {
                window.clearInterval(pollTimer);
                pollTimer = undefined;
            }
        }
        else {
            setMessage(statusMessage, "处理中，请稍候...");
        }
        return status;
    }
    catch (err) {
        const message = err instanceof Error ? err.message : "状态查询失败";
        setMessage(statusMessage, message, true);
        return null;
    }
};
const uploadFile = async (file) => {
    var _a, _b;
    setMessage(uploadHint, `正在上传：${file.name}`);
    try {
        const formData = new FormData();
        formData.append("file", file);
        const res = await fetchWithAutoRefresh("/api/tasks", {
            method: "POST",
            body: formData,
        });
        if (!res.ok) {
            if (res.status === 401) {
                throw new Error("登录已过期，请重新登录。");
            }
            throw new Error(`上传失败: ${res.status}`);
        }
        const data = await res.json();
        const taskId = (_b = (_a = data.task_id) !== null && _a !== void 0 ? _a : data.id) !== null && _b !== void 0 ? _b : data.taskId;
        if (!taskId) {
            throw new Error("返回中未找到 task_id");
        }
        setCurrentTask(taskId);
        startPolling(taskId);
        if (currentUser) {
            void loadTaskHistory();
        }
        setMessage(uploadHint, "上传成功，开始处理。");
    }
    catch (err) {
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
    var _a, _b;
    event.preventDefault();
    uploadZone.classList.remove("dragover");
    handleFiles((_b = (_a = event.dataTransfer) === null || _a === void 0 ? void 0 : _a.files) !== null && _b !== void 0 ? _b : null);
});
logoutButton.addEventListener("click", () => {
    void handleLogout();
});
historyRefreshButton.addEventListener("click", () => {
    void loadTaskHistory();
});
selectButton.addEventListener("click", () => {
    fileInput.click();
});
fileInput.addEventListener("change", () => {
    handleFiles(fileInput.files);
});
queryButton.addEventListener("click", async () => {
    const taskId = queryInput.value.trim();
    await runTaskQuery(taskId);
});
void loadCurrentUser();
export {};
