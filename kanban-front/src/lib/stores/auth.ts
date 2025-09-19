import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

// Define the user type
type User = {
  userId: string;
  username: string;
} | null;

// Create the auth store with initial state
function createAuthStore() {
  const { subscribe, set, update } = writable<{
    user: User;
    token: string | null;
    isLoading: boolean;
  }>({
    user: null,
    token: null,
    isLoading: true
  });

  // Initialize from localStorage when in browser
  if (browser) {
    const token = localStorage.getItem('token');
    const userStr = localStorage.getItem('user');
    let user = null;
    
    try {
      if (userStr) {
        user = JSON.parse(userStr);
      }
    } catch (e) {
      console.error('Failed to parse user from localStorage');
    }
    
    set({
      user,
      token,
      isLoading: false
    });
  }

  return {
    subscribe,
    login: (user: User, token: string) => {
      if (browser) {
        localStorage.setItem('token', token);
        localStorage.setItem('user', JSON.stringify(user));
      }
      
      set({
        user,
        token,
        isLoading: false
      });
    },
    logout: () => {
      if (browser) {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
      }
      
      set({
        user: null,
        token: null,
        isLoading: false
      });
    },
    initialize: () => {
      update(state => ({ ...state, isLoading: false }));
    }
  };
}

export const auth = createAuthStore();
export const isAuthenticated = derived(auth, $auth => !!$auth.token);
export const isLoading = derived(auth, $auth => $auth.isLoading);