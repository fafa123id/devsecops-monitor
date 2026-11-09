export default defineNuxtRouteMiddleware(async () => {
  const { checkAuth } = useAuth();

  try {
    const user = await checkAuth(true);

    if (user) {
      return navigateTo("/", {
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