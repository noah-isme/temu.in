
import { create } from 'zustand';
import { me as apiMe } from '../api';
import { logout as apiLogout, clearAuthToken } from '../api';

type User = {
  id: number;
  email: string;
  role: string;
  name?: string;
} | null;

interface AppState {
  initialized: boolean;
  user: User;
  initialize: () => Promise<void>;
  setUser: (u: User) => void;
  logout: () => Promise<void>;
}

export const useAppStore = create<AppState>((set) => ({
  initialized: false,
  user: null,
  setUser: (u: User) => set({ user: u }),
  initialize: async () => {
    try {
      const data = await apiMe();
      set({ user: data.user ?? null, initialized: true });
    } catch (err) {
      set({ user: null, initialized: true });
    }
  }
  ,
  logout: async () => {
    try {
      await apiLogout();
    } catch (e) {
      // swallow
    }
    clearAuthToken();
    set({ user: null });
  }
}));
