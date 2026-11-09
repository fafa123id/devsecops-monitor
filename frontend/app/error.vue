
<script setup lang="ts">
import type { NuxtError } from "#app";

const props = defineProps<{
  error: NuxtError;
}>();

const isNotFound = computed(() => props.error.statusCode === 404);

const handleGoHome = () => {
  clearError({
    redirect: "/",
  });
};
</script>

<template>
  <main class="error-page">
    <div class="ambient ambient-one"></div>
    <div class="ambient ambient-two"></div>

    <section class="error-container">
      <!-- Brand -->
      <div class="brand">
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

        <h2>
          DevSecOps<span>Monitor</span>
        </h2>
      </div>

      <!-- Error content -->
      <div class="error-card">
        <div class="error-icon">
          <svg
            v-if="isNotFound"
            viewBox="0 0 24 24"
            fill="none"
          >
            <circle
              cx="11"
              cy="11"
              r="7"
              stroke="currentColor"
              stroke-width="1.7"
            />
            <path
              d="m16 16 4 4M8.5 8.5l5 5m0-5-5 5"
              stroke="currentColor"
              stroke-width="1.7"
              stroke-linecap="round"
            />
          </svg>

          <svg
            v-else
            viewBox="0 0 24 24"
            fill="none"
          >
            <path
              d="M12 3 21 19H3L12 3Z"
              stroke="currentColor"
              stroke-width="1.7"
              stroke-linejoin="round"
            />
            <path
              d="M12 9v4m0 3h.01"
              stroke="currentColor"
              stroke-width="1.7"
              stroke-linecap="round"
            />
          </svg>
        </div>

        <div class="error-status">
          ERROR CODE {{ error.statusCode || 500 }}
        </div>

        <h1>
          {{ isNotFound ? "Page not found" : "Something went wrong" }}
        </h1>

        <p>
          {{
            isNotFound
              ? "The page you're looking for doesn't exist or may have been moved."
              : "An unexpected error occurred while processing your request."
          }}
        </p>

        <div class="error-divider"></div>

        <div class="error-detail">
          <span>REQUEST STATUS</span>
          <span
            :class="isNotFound ? 'status-warning' : 'status-danger'"
          >
            {{ error.statusCode || 500 }}
            {{ isNotFound ? "Not Found" : "Server Error" }}
          </span>
        </div>

        <button
          type="button"
          class="home-button"
          @click="handleGoHome"
        >
          <svg viewBox="0 0 24 24" fill="none">
            <path
              d="m10 17-5-5 5-5m-5 5h14"
              stroke="currentColor"
              stroke-width="1.7"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>

          Back to dashboard
        </button>
      </div>

      <footer>
        DEVSECOPS MONITOR
        <span>·</span>
        SECURITY OPERATIONS PLATFORM
      </footer>
    </section>
  </main>
</template>

<style scoped>
.error-page {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  min-height: 100svh;
  width: 100%;
  padding: 40px 20px;
  overflow: hidden;
  isolation: isolate;
  background: #090d16;
  color: #f1f5f9;
  font-family: Inter, ui-sans-serif, system-ui, sans-serif;
}

.error-page *,
.error-page *::before,
.error-page *::after {
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
  top: -250px;
  left: calc(50% - 450px);
  background: rgba(14, 165, 233, 0.12);
}

.ambient-two {
  bottom: -280px;
  right: calc(50% - 450px);
  background: rgba(59, 130, 246, 0.10);
}

.error-container {
  width: 100%;
  max-width: 440px;
  margin: auto;
}

.brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 32px;
  text-align: center;
}

.brand-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  margin: 0;
  border: 1px solid rgba(56, 189, 248, 0.25);
  border-radius: 13px;
  background: rgba(14, 165, 233, 0.10);
  color: #38bdf8;
}

.brand-icon svg {
  width: 23px;
  height: 23px;
}

.brand h2 {
  margin: 0;
  color: #f8fafc;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.4px;
}

.brand h2 span {
  color: #38bdf8;
}

.error-card {
  padding: 36px;
  border: 1px solid #263247;
  border-radius: 17px;
  background: rgba(17, 24, 39, 0.96);
  box-shadow: 0 25px 70px rgba(0, 0, 0, 0.28);
  text-align: center;
}

.error-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 58px;
  height: 58px;
  margin: 0 auto 23px;
  border: 1px solid rgba(56, 189, 248, 0.20);
  border-radius: 16px;
  background: rgba(14, 165, 233, 0.09);
  color: #38bdf8;
}

.error-icon svg {
  width: 28px;
  height: 28px;
}

.error-status {
  color: #38bdf8;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 1.6px;
}

.error-card h1 {
  margin: 13px 0 10px;
  color: #f8fafc;
  font-size: 25px;
  font-weight: 700;
  letter-spacing: -0.8px;
}

.error-card > p {
  margin: 0 auto;
  max-width: 310px;
  color: #94a3b8;
  font-size: 13px;
  line-height: 1.7;
}

.error-divider {
  height: 1px;
  margin: 27px 0 20px;
  background: #263247;
}

.error-detail {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 25px;
}

.error-detail > span:first-child {
  color: #64748b;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 1px;
}

.error-detail > span:last-child {
  font-size: 11px;
  font-weight: 600;
}

.status-warning {
  color: #fbbf24;
}

.status-danger {
  color: #f87171;
}

.home-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  width: 100%;
  height: 46px;
  padding: 0 16px;
  border: 0;
  border-radius: 9px;
  background: #0ea5e9;
  color: #06111d;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 700;
  transition: background 160ms, transform 160ms;
}

.home-button:hover {
  background: #38bdf8;
  transform: translateY(-1px);
}

.home-button svg {
  width: 17px;
  height: 17px;
}

.error-container footer {
  margin-top: 26px;
  color: #64748b;
  font-size: 9px;
  letter-spacing: 1.1px;
  text-align: center;
}

.error-container footer span {
  margin: 0 7px;
  color: #38bdf8;
}

@media (max-width: 480px) {
  .error-card {
    padding: 28px 22px;
  }

  .error-card h1 {
    font-size: 22px;
  }
}
</style>
