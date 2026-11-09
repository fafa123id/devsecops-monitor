<template>
  <div class="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#090d16] px-5 py-12">
    <!-- Background decoration -->
    <div class="pointer-events-none absolute -left-40 -top-40 h-96 w-96 rounded-full bg-sky-500/10 blur-[120px]"></div>
    <div class="pointer-events-none absolute -bottom-40 -right-40 h-96 w-96 rounded-full bg-blue-500/10 blur-[120px]"></div>

    <main class="relative z-10 w-full max-w-md">
      <slot />
    </main>

    <div class="absolute bottom-6 text-center text-xs text-slate-600">
      DevSecOps Monitor &copy; {{ new Date().getFullYear() }}
    </div>
  </div>
</template>