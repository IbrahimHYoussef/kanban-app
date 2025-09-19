import { goto } from '$app/navigation';
import { auth, isAuthenticated, isLoading } from '$lib/stores/auth';
import { get } from 'svelte/store';

// Define route groups
const PUBLIC_ROUTES = ['/login', '/register'];
const PRIVATE_ROUTES = ['/dashboard', '/projects'];

// Middleware runs before page navigation
export async function handleNavigate({ from, to }: { from: URL | null, to: URL }) {
  // Wait until auth store is initialized
  if (get(isLoading)) {
    await new Promise<void>(resolve => {
      const unsubscribe = isLoading.subscribe(loading => {
        if (!loading) {
          unsubscribe();
          resolve();
        }
      });
    });
  }
  
  const authenticated = get(isAuthenticated);
  
  // Check if trying to access private route without auth
  if (!authenticated && isPrivateRoute(to.pathname)) {
    goto('/login');
    return false; // Prevent navigation
  }
  
  // Check if trying to access auth routes when already authenticated
  if (authenticated && isAuthRoute(to.pathname)) {
    goto('/dashboard');
    return false; // Prevent navigation
  }
  
  return true; // Allow navigation
}

// Helper functions to check route type
function isPrivateRoute(pathname: string): boolean {
  return PRIVATE_ROUTES.some(route => pathname.startsWith(route));
}

function isAuthRoute(pathname: string): boolean {
  return PUBLIC_ROUTES.includes(pathname);
}