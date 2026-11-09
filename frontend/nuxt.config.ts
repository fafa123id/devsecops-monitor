export default defineNuxtConfig({
  modules: ["@nuxtjs/tailwindcss"],
  css: ["~/assets/css/main.css"],

  runtimeConfig: {
    public: {
      apiBase: "http://localhost:18080",
    },
  },

  compatibilityDate: "2025-07-15",
  devtools: { enabled: true },
});