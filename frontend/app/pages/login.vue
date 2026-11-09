
<script setup lang="ts">
definePageMeta({
  layout: false,
  middleware: "guest",
});

const { login, loading } = useAuth();

const form = reactive({
  username: "",
  password: "",
});

const errorMessage = ref("");
const showPassword = ref(false);

async function handleLogin() {
  errorMessage.value = "";

  try {
    await login({
      username: form.username.trim(),
      password: form.password,
    });

    await navigateTo("/", { replace: true });
  } catch (error: unknown) {
    const apiError = error as {
      data?: { message?: string };
      statusMessage?: string;
    };

    errorMessage.value =
      apiError.data?.message ||
      apiError.statusMessage ||
      "Unable to sign in. Please try again.";
  }
}
</script>

<template>
  <div class="login-page">
    <div class="ambient ambient-one"></div>
    <div class="ambient ambient-two"></div>

    <main class="login-wrapper">
      <header class="brand">
        <div class="brand-icon">
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="M12 3 20 6V11C20 16 16.8 19.4 12 21C7.2 19.4 4 16 4 11V6L12 3Z"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linejoin="round"
            />
            <path
              d="M9 12L11 14L15.5 9.5"
              stroke="currentColor"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>

        <h1>DevSecOps Monitor</h1>
        <p>Application Security Monitoring Platform</p>
      </header>

      <section class="login-card">
        <div class="card-heading">
          <div class="eyebrow">
            <span class="status-dot"></span>
            SECURE ACCESS
          </div>

          <h2>Welcome back</h2>
          <p>Sign in to continue to your security dashboard.</p>
        </div>

        <form @submit.prevent="handleLogin">
          <div class="form-field">
            <label for="username">Username</label>
            <input
              id="username"
              v-model="form.username"
              type="text"
              autocomplete="username"
              placeholder="Enter your username"
              required
              autofocus
            />
          </div>

          <div class="form-field">
            <label for="password">Password</label>

            <div class="password-wrapper">
              <input
                id="password"
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                placeholder="Enter your password"
                required
              />

              <button
                type="button"
                class="password-toggle"
                @click="showPassword = !showPassword"
              >
                {{ showPassword ? "Hide" : "Show" }}
              </button>
            </div>
          </div>

          <div v-if="errorMessage" class="error-message" role="alert">
            <span>!</span>
            {{ errorMessage }}
          </div>

          <button
            type="submit"
            class="submit-button"
            :disabled="loading || !form.username.trim() || !form.password"
          >
            <svg
              v-if="loading"
              class="spinner"
              viewBox="0 0 24 24"
              fill="none"
            >
              <circle
                cx="12"
                cy="12"
                r="9"
                stroke="currentColor"
                stroke-width="3"
                opacity=".25"
              />
              <path
                d="M21 12a9 9 0 0 0-9-9"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
              />
            </svg>

            {{ loading ? "Signing in..." : "Sign in to dashboard" }}

            <span v-if="!loading" class="button-arrow">→</span>
          </button>
        </form>

        <div class="card-footer">
          <svg viewBox="0 0 24 24" fill="none">
            <rect
              x="5"
              y="10"
              width="14"
              height="11"
              rx="2"
              stroke="currentColor"
              stroke-width="1.5"
            />
            <path
              d="M8 10V7a4 4 0 0 1 8 0v3"
              stroke="currentColor"
              stroke-width="1.5"
            />
          </svg>
          Your session is protected by secure authentication.
        </div>
      </section>

      <footer class="page-footer">
        <span>DevSecOps Monitor</span>
        <span class="footer-separator">·</span>
        Restricted access. Authorized personnel only.
      </footer>
    </main>
  </div>
</template>

<style>
.login-page {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  min-height: 100svh;
  width: 100%;
  padding: 48px 24px;
  overflow: hidden;
  isolation: isolate;
  background: #090d16;
  color: #f1f5f9;
  font-family: Inter, ui-sans-serif, system-ui, sans-serif;
}

.login-page *,
.login-page *::before,
.login-page *::after {
  box-sizing: border-box;
}

.ambient {
  position: absolute;
  z-index: -1;
  width: 420px;
  height: 420px;
  border-radius: 50%;
  filter: blur(120px);
  pointer-events: none;
}

.ambient-one {
  top: -240px;
  left: calc(50% - 430px);
  background: rgba(14, 165, 233, 0.13);
}

.ambient-two {
  bottom: -280px;
  right: calc(50% - 430px);
  background: rgba(59, 130, 246, 0.10);
}

.login-wrapper {
  width: 100%;
  max-width: 430px;
  margin: auto;
  position: relative;
}

.brand {
  text-align: center;
  margin-bottom: 34px;
}

.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 58px;
  height: 58px;
  margin: 0 auto 17px;
  border: 1px solid rgba(56, 189, 248, 0.25);
  border-radius: 17px;
  background: rgba(14, 165, 233, 0.10);
  color: #38bdf8;
}

.brand-icon svg {
  width: 29px;
  height: 29px;
}

.brand h1 {
  margin: 0;
  color: #f8fafc;
  font-size: 25px;
  font-weight: 700;
  letter-spacing: -0.8px;
}

.brand p {
  margin: 9px 0 0;
  color: #8293aa;
  font-size: 13px;
}

.login-card {
  padding: 34px;
  border: 1px solid #263247;
  border-radius: 17px;
  background: rgba(17, 24, 39, 0.96);
  box-shadow:
    0 25px 70px rgba(0, 0, 0, 0.28),
    0 1px 0 rgba(255, 255, 255, 0.025) inset;
}

.card-heading {
  margin-bottom: 29px;
}

.eyebrow {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 17px;
  color: #38bdf8;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 1.5px;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #38bdf8;
  box-shadow: 0 0 10px rgba(56, 189, 248, 0.55);
}

.card-heading h2 {
  margin: 0;
  color: #f8fafc;
  font-size: 23px;
  font-weight: 650;
  letter-spacing: -0.5px;
}

.card-heading p {
  margin: 9px 0 0;
  color: #94a3b8;
  font-size: 13px;
  line-height: 1.6;
}

.form-field {
  margin-bottom: 21px;
}

.form-field label {
  display: block;
  margin-bottom: 9px;
  color: #d1d9e6;
  font-size: 13px;
  font-weight: 550;
}

.form-field input {
  display: block;
  width: 100%;
  height: 46px;
  padding: 0 14px;
  border: 1px solid #344258;
  border-radius: 9px;
  outline: none;
  background: #0b1220;
  color: #f8fafc;
  font-family: inherit;
  font-size: 13px;
  transition: border-color 160ms, box-shadow 160ms;
}

.form-field input::placeholder {
  color: #607089;
}

.form-field input:focus {
  border-color: #38bdf8;
  box-shadow: 0 0 0 3px rgba(56, 189, 248, 0.10);
}

.password-wrapper {
  position: relative;
}

.password-wrapper input {
  padding-right: 65px;
}

.password-toggle {
  position: absolute;
  top: 50%;
  right: 13px;
  transform: translateY(-50%);
  padding: 5px;
  border: 0;
  background: transparent;
  color: #8293aa;
  cursor: pointer;
  font-size: 11px;
}

.password-toggle:hover {
  color: #38bdf8;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
  padding: 12px 13px;
  border: 1px solid rgba(248, 113, 113, 0.25);
  border-radius: 9px;
  background: rgba(127, 29, 29, 0.15);
  color: #fca5a5;
  font-size: 12px;
}

.error-message span {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 17px;
  height: 17px;
  flex-shrink: 0;
  border-radius: 50%;
  background: rgba(248, 113, 113, 0.15);
  font-weight: 700;
}

.submit-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  height: 47px;
  margin-top: 27px;
  padding: 0 16px;
  border: 0;
  border-radius: 9px;
  background: #0ea5e9;
  color: #06111d;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 700;
  transition: background 160ms, transform 160ms, box-shadow 160ms;
}

.submit-button:hover:not(:disabled) {
  background: #38bdf8;
  box-shadow: 0 5px 22px rgba(14, 165, 233, 0.17);
  transform: translateY(-1px);
}

.submit-button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.button-arrow {
  font-size: 18px;
  font-weight: 400;
}

.spinner {
  width: 16px;
  height: 16px;
  animation: spin 0.8s linear infinite;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 25px;
  padding-top: 20px;
  border-top: 1px solid #263247;
  color: #718198;
  font-size: 10px;
  text-align: center;
}

.card-footer svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.page-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  margin-top: 26px;
  color: #64748b;
  font-size: 10px;
  text-align: center;
}

.footer-separator {
  color: #475569;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 480px) {
  .login-page {
    padding: 32px 18px;
  }

  .login-card {
    padding: 25px 21px;
  }

  .brand {
    margin-bottom: 27px;
  }

  .brand h1 {
    font-size: 22px;
  }

  .page-footer {
    flex-wrap: wrap;
    line-height: 1.6;
  }
}
</style>
