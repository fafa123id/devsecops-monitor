export default defineNuxtConfig({
  modules: ["@nuxtjs/tailwindcss"],
  css: ["~/assets/css/main.css"],

  runtimeConfig: {
    apiServerBase: process.env.NUXT_API_SERVER_BASE || "http://api:8000",

    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || "http://localhost:18080",
    },
  },

  compatibilityDate: "2025-07-15",
  devtools: { enabled: true },
});
