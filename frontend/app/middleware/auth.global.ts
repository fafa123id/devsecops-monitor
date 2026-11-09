export default defineNuxtRouteMiddleware(async (to) => {

  if (to.matched.length === 0) {
    return;
  }

  if (to.path === "/login") {
    return;
  }

  const { checkAuth } = useAuth();

  try {
    const user = await checkAuth();

    if (!user) {
      return navigateTo("/login", {
        replace: true,
      });
    }
  } catch (error) {
    console.error("Authentication check failed:", error);

    throw createError({
      statusCode: 503,
      statusMessage: "Unable to connect to authentication service.",
    });
  }
});