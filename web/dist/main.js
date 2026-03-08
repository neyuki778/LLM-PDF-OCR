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
let pollTimer;
let refreshPromise = null;
const setMessage = (el, message, isError = false) => {
    el.textContent = message;
    el.classList.toggle("error", isError);
};
const setCurrentUser = (user) => {
    if (user) {
        authUserEmail.textContent = user.email;
        authLink.classList.add("hidden");
        authUser.classList.remove("hidden");
        return;
    }
    authUserEmail.textContent = "";
    authUser.classList.add("hidden");
    authLink.classList.remove("hidden");
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
const normalizeStatus = (status) => status.toLowerCase();
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
void loadCurrentUser();
export {};
