const authEmailInput = document.getElementById("auth-email");
const authPasswordInput = document.getElementById("auth-password");
const registerButton = document.getElementById("register-button");
const loginButton = document.getElementById("login-button");
const authMessage = document.getElementById("auth-message");
const authUser = document.getElementById("auth-user");
const authUserEmail = document.getElementById("auth-user-email");
const logoutButton = document.getElementById("logout-button");
let refreshPromise = null;
const setMessage = (message, isError = false) => {
    authMessage.textContent = message;
    authMessage.classList.toggle("error", isError);
};
const parseErrorMessage = async (res, fallback) => {
    try {
        const data = await res.json();
        if (typeof (data === null || data === void 0 ? void 0 : data.error) === "string" && data.error.trim() !== "") {
            return data.error;
        }
    }
    catch (_) {
        // ignore parse error
    }
    return fallback;
};
const setCurrentUser = (user) => {
    if (user) {
        authUserEmail.textContent = user.email;
        authUser.classList.remove("hidden");
        return;
    }
    authUserEmail.textContent = "";
    authUser.classList.add("hidden");
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
const getCredentials = () => {
    const email = authEmailInput.value.trim();
    const password = authPasswordInput.value;
    if (!email) {
        setMessage("请输入邮箱。", true);
        return null;
    }
    if (!password) {
        setMessage("请输入密码。", true);
        return null;
    }
    return { email, password };
};
const handleRegister = async () => {
    const credentials = getCredentials();
    if (!credentials) {
        return;
    }
    setMessage("正在注册...");
    try {
        const res = await fetch("/api/auth/register", {
            method: "POST",
            credentials: "include",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(credentials),
        });
        if (!res.ok) {
            const message = await parseErrorMessage(res, `注册失败: ${res.status}`);
            setMessage(message, true);
            return;
        }
        authPasswordInput.value = "";
        setMessage("注册成功，请登录。");
    }
    catch (_) {
        setMessage("网络错误，注册失败。", true);
    }
};
const handleLogin = async () => {
    const credentials = getCredentials();
    if (!credentials) {
        return;
    }
    setMessage("正在登录...");
    try {
        const res = await fetch("/api/auth/login", {
            method: "POST",
            credentials: "include",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(credentials),
        });
        if (!res.ok) {
            const message = await parseErrorMessage(res, `登录失败: ${res.status}`);
            setMessage(message, true);
            return;
        }
        setMessage("登录成功，正在返回首页...");
        window.location.href = "/";
    }
    catch (_) {
        setMessage("网络错误，登录失败。", true);
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
    setMessage("已登出。");
};
registerButton.addEventListener("click", () => {
    void handleRegister();
});
loginButton.addEventListener("click", () => {
    void handleLogin();
});
logoutButton.addEventListener("click", () => {
    void handleLogout();
});
authPasswordInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
        event.preventDefault();
        void handleLogin();
    }
});
void loadCurrentUser();
export {};
