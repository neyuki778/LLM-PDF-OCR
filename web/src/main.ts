export {}

type TaskStatus = "pending" | "processing" | "completed" | "failed" | string
type CurrentUser = {
  id: string
  email: string
}

const uploadZone = document.getElementById("upload-zone") as HTMLDivElement
const selectButton = document.getElementById("select-button") as HTMLButtonElement
const fileInput = document.getElementById("file-input") as HTMLInputElement
const uploadHint = document.getElementById("upload-hint") as HTMLParagraphElement
const taskIdEl = document.getElementById("task-id") as HTMLSpanElement
const statusEl = document.getElementById("task-status") as HTMLSpanElement
const progressEl = document.getElementById("task-progress") as HTMLSpanElement
const statusMessage = document.getElementById("status-message") as HTMLParagraphElement
const resultLink = document.getElementById("result-link") as HTMLAnchorElement
const queryInput = document.getElementById("query-input") as HTMLInputElement
const queryButton = document.getElementById("query-button") as HTMLButtonElement
const queryMessage = document.getElementById("query-message") as HTMLParagraphElement
const authLink = document.getElementById("auth-link") as HTMLAnchorElement
const authUser = document.getElementById("auth-user") as HTMLDivElement
const authUserEmail = document.getElementById("auth-user-email") as HTMLSpanElement
const logoutButton = document.getElementById("logout-button") as HTMLButtonElement

let pollTimer: number | undefined
let refreshPromise: Promise<boolean> | null = null

const setMessage = (el: HTMLParagraphElement, message: string, isError = false) => {
  el.textContent = message
  el.classList.toggle("error", isError)
}

const setCurrentUser = (user: CurrentUser | null) => {
  if (user) {
    authUserEmail.textContent = user.email
    authLink.classList.add("hidden")
    authUser.classList.remove("hidden")
    return
  }
  authUserEmail.textContent = ""
  authUser.classList.add("hidden")
  authLink.classList.remove("hidden")
}

const refreshSessionSilently = async () => {
  if (refreshPromise) {
    return refreshPromise
  }
  refreshPromise = (async () => {
    try {
      const res = await fetch("/api/auth/refresh", {
        method: "POST",
        credentials: "include",
      })
      if (!res.ok) {
        setCurrentUser(null)
        return false
      }
      return true
    } catch (_) {
      setCurrentUser(null)
      return false
    }
  })().finally(() => {
    refreshPromise = null
  })
  return refreshPromise
}

const fetchWithAutoRefresh = async (url: string, init: RequestInit = {}, canRetry = true) => {
  const res = await fetch(url, {
    ...init,
    credentials: "include",
  })
  if (res.status === 401 && canRetry && url !== "/api/auth/refresh") {
    const refreshed = await refreshSessionSilently()
    if (refreshed) {
      return fetchWithAutoRefresh(url, init, false)
    }
  }
  return res
}

const loadCurrentUser = async () => {
  try {
    const res = await fetchWithAutoRefresh("/api/auth/me")
    if (!res.ok) {
      setCurrentUser(null)
      return
    }
    const data = await res.json()
    const user = data?.user as CurrentUser | undefined
    if (!user || typeof user.email !== "string") {
      setCurrentUser(null)
      return
    }
    setCurrentUser(user)
  } catch (_) {
    setCurrentUser(null)
  }
}

const handleLogout = async () => {
  try {
    await fetch("/api/auth/logout", {
      method: "POST",
      credentials: "include",
    })
  } catch (_) {
    // ignore network error
  }
  setCurrentUser(null)
}

const setCurrentTask = (taskId: string) => {
  taskIdEl.textContent = taskId
  statusEl.textContent = "处理中"
  progressEl.textContent = "-"
  resultLink.classList.add("hidden")
  resultLink.href = "#"
  resultLink.removeAttribute("download")
  setMessage(statusMessage, "")
}

const startPolling = (taskId: string) => {
  if (pollTimer) {
    window.clearInterval(pollTimer)
  }
  pollTimer = window.setInterval(() => {
    void fetchStatus(taskId)
  }, 2000)
}

const normalizeStatus = (status: TaskStatus) => status.toLowerCase()

const fetchStatus = async (taskId: string) => {
  try {
    const res = await fetchWithAutoRefresh(`/api/tasks/${encodeURIComponent(taskId)}`)
    if (!res.ok) {
      if (res.status === 401) {
        throw new Error("登录已过期，请重新登录。")
      }
      throw new Error(`状态查询失败: ${res.status}`)
    }
    const data = await res.json()
    const status = normalizeStatus(data.status ?? data.state ?? "unknown")
    const progress = data.completed_count ?? data.progress ?? ""
    statusEl.textContent = status
    progressEl.textContent = progress || "-"

    if (status === "completed" || status === "success" || status === "done") {
      resultLink.href = `/api/tasks/${encodeURIComponent(taskId)}/result`
      resultLink.setAttribute("download", `${taskId}.md`)
      resultLink.classList.remove("hidden")
      setMessage(statusMessage, "任务完成，可以下载结果。")
      if (pollTimer) {
        window.clearInterval(pollTimer)
        pollTimer = undefined
      }
    } else if (status === "failed") {
      setMessage(statusMessage, data.error ?? "任务失败。", true)
      if (pollTimer) {
        window.clearInterval(pollTimer)
        pollTimer = undefined
      }
    } else {
      setMessage(statusMessage, "处理中，请稍候...")
    }
    return status
  } catch (err) {
    const message = err instanceof Error ? err.message : "状态查询失败"
    setMessage(statusMessage, message, true)
    return null
  }
}

const uploadFile = async (file: File) => {
  setMessage(uploadHint, `正在上传：${file.name}`)
  try {
    const formData = new FormData()
    formData.append("file", file)
    const res = await fetchWithAutoRefresh("/api/tasks", {
      method: "POST",
      body: formData,
    })
    if (!res.ok) {
      if (res.status === 401) {
        throw new Error("登录已过期，请重新登录。")
      }
      throw new Error(`上传失败: ${res.status}`)
    }
    const data = await res.json()
    const taskId = data.task_id ?? data.id ?? data.taskId
    if (!taskId) {
      throw new Error("返回中未找到 task_id")
    }
    setCurrentTask(taskId)
    startPolling(taskId)
    setMessage(uploadHint, "上传成功，开始处理。")
  } catch (err) {
    const message = err instanceof Error ? err.message : "上传失败"
    setMessage(uploadHint, message, true)
  }
}

const handleFiles = (files: FileList | null) => {
  if (!files || files.length === 0) {
    return
  }
  const file = files[0]
  if (file.type !== "application/pdf") {
    setMessage(uploadHint, "请选择 PDF 文件。", true)
    return
  }
  void uploadFile(file)
}

uploadZone.addEventListener("dragover", (event) => {
  event.preventDefault()
  uploadZone.classList.add("dragover")
})

uploadZone.addEventListener("dragleave", () => {
  uploadZone.classList.remove("dragover")
})

uploadZone.addEventListener("drop", (event) => {
  event.preventDefault()
  uploadZone.classList.remove("dragover")
  handleFiles(event.dataTransfer?.files ?? null)
})

logoutButton.addEventListener("click", () => {
  void handleLogout()
})

selectButton.addEventListener("click", () => {
  fileInput.click()
})

fileInput.addEventListener("change", () => {
  handleFiles(fileInput.files)
})

queryButton.addEventListener("click", async () => {
  const taskId = queryInput.value.trim()
  if (!taskId) {
    setMessage(queryMessage, "请输入 Task ID。", true)
    return
  }
  setMessage(queryMessage, "")
  setCurrentTask(taskId)
  const status = await fetchStatus(taskId)
  if (status && status !== "completed" && status !== "success" && status !== "done" && status !== "failed") {
    startPolling(taskId)
  }
})

void loadCurrentUser()
