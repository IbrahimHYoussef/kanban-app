<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { auth, isLoading } from '$lib/stores/auth';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  
  let { children } = $props();
  
  // Protected routes that require authentication
  const PRIVATE_ROUTES = ['/dashboard', '/projects'];
  // Public routes for authentication
  const PUBLIC_ROUTES = ['/login', '/register'];
  
  onMount(() => {
    // Check if user is already authenticated on mount
    auth.initialize();
    
    // Handle initial route based on auth state
    return isLoading.subscribe(loading => {
      if (!loading) {
        const pathname = $page.url.pathname;
        const isPrivate = PRIVATE_ROUTES.some(route => pathname.startsWith(route));
        const isPublic = PUBLIC_ROUTES.includes(pathname);
        
        // If user is on private route but not logged in, redirect to login
        if (isPrivate && !$auth.token) {
          goto('/login');
        }
        
        // If user is on auth route but already logged in, redirect to dashboard
        if (isPublic && $auth.token) {
          goto('/dashboard');
        }
      }
    });
  });
</script>

{#if $isLoading}
  <div class="min-h-screen flex justify-center items-center">
    <div class="animate-spin h-12 w-12 border-4 border-indigo-500 rounded-full border-t-transparent"></div>
  </div>
{:else}
  {@render children()}
{/if}
