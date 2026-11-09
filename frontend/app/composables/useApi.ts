export const useApi = () => {
  const config = useRuntimeConfig();
  const baseURL = config.public.apiBase as string;

  const request = async <T>(
    path: string,
    method: "GET" | "POST",
    body?: Record<string, any>,
  ): Promise<T> => {
    const headers: Record<string, string> = {
      Accept: "application/json",
      "X-Requested-With": "XMLHttpRequest",
    };

    if (import.meta.server) {
      const requestHeaders = useRequestHeaders(["cookie"]);
      const requestUrl = useRequestURL();

      if (requestHeaders.cookie) {
        headers.cookie = requestHeaders.cookie;
      }

      headers.Origin = requestUrl.origin;
    }

    if (method === "POST") {
      await $fetch("/sanctum/csrf-cookie", {
        baseURL,
        credentials: "include",
      });

      if (import.meta.client) {
        const cookie = document.cookie
          .split("; ")
          .find((item) => item.startsWith("XSRF-TOKEN="));

        if (cookie) {
          headers["X-XSRF-TOKEN"] = decodeURIComponent(
            cookie.substring("XSRF-TOKEN=".length),
          );
        }
      }
    }

    return await $fetch<T>(path, {
      baseURL,
      method,
      body,
      headers,
      credentials: "include",
    });
  };

  return {
    get: <T>(path: string) => request<T>(path, "GET"),

    post: <T>(path: string, body?: Record<string, any>) =>
      request<T>(path, "POST", body),
  };
};
