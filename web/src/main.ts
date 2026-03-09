export {}

type TaskStatus = "pending" | "processing" | "completed" | "failed" | string
type CurrentUser = {
  id: string
  email: string
}
type HistoryTaskItem = {
  task_id: string
  status: string
  created_at: number
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
const authBenefitText = document.getElementById("auth-benefit-text") as HTMLParagraphElement
const historyCard = document.getElementById("history-card") as HTMLDivElement
const historyHint = document.getElementById("history-hint") as HTMLParagraphElement
const historyList = document.getElementById("history-list") as HTMLUListElement
const historyRefreshButton = document.getElementById("history-refresh-button") as HTMLButtonElement

let pollTimer: number | undefined
let refreshPromise: Promise<boolean> | null = null
let currentUser: CurrentUser | null = null

const setMessage = (el: HTMLParagraphElement, message: string, isError = false) => {
  el.textContent = message
  el.classList.toggle("error", isError)
}

const setAuthBenefit = (user: CurrentUser | null) => {
  if (user) {
    authBenefitText.textContent = "已登录：历史任务会自动保存。登录用户单次额度更高（最多 40 页），并可使用额外功能。"
    return
  }
  authBenefitText.textContent = "游客模式不保存历史任务记录。登录 / 注册后可获得更高额度（单次最多 40 页）并使用历史任务等额外功能。"
}

const clearHistoryList = () => {
  historyList.innerHTML = ""
}

const formatStatus = (status: string) => {
  const normalized = status.toLowerCase()
  if (normalized === "completed") return "已完成"
  if (normalized === "processing") return "处理中"
  if (normalized === "pending") return "排队中"
  if (normalized === "failed") return "失败"
  return status
}

const formatUnixTime = (sec: number) => {
  if (!Number.isFinite(sec) || sec <= 0) {
    return "-"
  }
  return new Date(sec * 1000).toLocaleString("zh-CN", { hour12: false })
}

const normalizeStatus = (status: TaskStatus) => status.toLowerCase()

const runTaskQuery = async (taskId: string) => {
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
}

const renderHistoryList = (items: HistoryTaskItem[]) => {
  clearHistoryList()
  for (const item of items) {
    const li = document.createElement("li")
    li.className = "history-item"

    const left = document.createElement("div")
    left.className = "history-main"

    const id = document.createElement("span")
    id.className = "history-id"
    id.textContent = item.task_id

    const meta = document.createElement("span")
    meta.className = "history-meta"
    meta.textContent = `${formatStatus(item.status)} · ${formatUnixTime(item.created_at)}`

    left.append(id, meta)

    const btn = document.createElement("button")
    btn.type = "button"
    btn.textContent = "查看"
    btn.addEventListener("click", () => {
      queryInput.value = item.task_id
      setMessage(queryMessage, "已从历史任务载入。")
      void runTaskQuery(item.task_id)
    })

    li.append(left, btn)
    historyList.appendChild(li)
  }
}

const loadTaskHistory = async () => {
  if (!currentUser) {
    clearHistoryList()
    setMessage(historyHint, "登录后可查看历史任务。")
    return
  }

  setMessage(historyHint, "正在加载历史任务...")
  try {
    const res = await fetchWithAutoRefresh("/api/tasks/history?limit=20")
    if (!res.ok) {
      if (res.status === 401) {
        setCurrentUser(null)
        return
      }
      throw new Error(`历史任务加载失败: ${res.status}`)
    }
    const data = (await res.json()) as { items?: HistoryTaskItem[] }
    const items = Array.isArray(data.items) ? data.items : []
    if (items.length === 0) {
      clearHistoryList()
      setMessage(historyHint, "暂无历史任务。")
      return
    }
    renderHistoryList(items)
    setMessage(historyHint, `已加载最近 ${items.length} 条历史任务。`)
  } catch (err) {
    const message = err instanceof Error ? err.message : "历史任务加载失败"
    clearHistoryList()
    setMessage(historyHint, message, true)
  }
}

const setCurrentUser = (user: CurrentUser | null) => {
  currentUser = user
  setAuthBenefit(user)
  if (user) {
    authUserEmail.textContent = user.email
    authLink.classList.add("hidden")
    authUser.classList.remove("hidden")
    historyCard.classList.remove("hidden")
    void loadTaskHistory()
    return
  }
  authUserEmail.textContent = ""
  authUser.classList.add("hidden")
  authLink.classList.remove("hidden")
  historyCard.classList.add("hidden")
  clearHistoryList()
  setMessage(historyHint, "登录后可查看历史任务。")
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
    if (currentUser) {
      void loadTaskHistory()
    }
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

historyRefreshButton.addEventListener("click", () => {
  void loadTaskHistory()
})

selectButton.addEventListener("click", () => {
  fileInput.click()
})

fileInput.addEventListener("change", () => {
  handleFiles(fileInput.files)
})

queryButton.addEventListener("click", async () => {
  const taskId = queryInput.value.trim()
  await runTaskQuery(taskId)
})

void loadCurrentUser()
