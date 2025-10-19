
import { create } from 'zustand';

interface AppState {
  initialized: boolean;
  initialize: () => void;
}

export const useAppStore = create<AppState>((set) => ({
  initialized: false,
  initialize: () =>
    set({
      initialized: true
    })
}));
