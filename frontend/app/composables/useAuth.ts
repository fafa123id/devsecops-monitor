interface AuthUser {
  id: number;
  name: string;
  username: string;
  email: string;
}

interface AuthResponse {
  message: string;
  user: AuthUser;
}

interface AuthCredentials {
  username: string;
  password: string;
}

export const useAuth = () => {
  const api = useApi();

  const user = useState<AuthUser | null>(
    "auth:user",
    () => null,
  );

  const initialized = useState(
    "auth:initialized",
    () => false,
  );

  const loading = useState(
    "auth:loading",
    () => false,
  );

  const isAuthenticated = computed(() => user.value !== null);

  function isUnauthorized(error: unknown): boolean {
    if (typeof error !== "object" || error === null) {
      return false;
    }

    const status = error as {
      status?: number;
      statusCode?: number;
    };

    return status.status === 401 || status.statusCode === 401;
  }

  async function checkAuth(force = false) {
    if (initialized.value && !force) {
      return user.value;
    }

    loading.value = true;

    try {
      const response = await api.get<{ user: AuthUser }>(
        "/api/auth/me",
      );

      user.value = response.user;
      initialized.value = true;

      return user.value;
    } catch (error) {
      if (isUnauthorized(error)) {
        user.value = null;
        initialized.value = true;

        return null;
      }

      throw error;
    } finally {
      loading.value = false;
    }
  }

  async function login(credentials: AuthCredentials) {
    loading.value = true;

    try {
      const response = await api.post<AuthResponse>(
        "/api/auth/login",
        credentials,
      );

      user.value = response.user;
      initialized.value = true;

      return response;
    } finally {
      loading.value = false;
    }
  }

  async function logout() {
    loading.value = true;

    try {
      await api.post<{ message: string }>(
        "/api/auth/logout",
      );

      user.value = null;
      initialized.value = true;
    } finally {
      loading.value = false;
    }
  }

  return {
    user,
    initialized,
    loading,
    isAuthenticated,
    checkAuth,
    login,
    logout,
  };
};