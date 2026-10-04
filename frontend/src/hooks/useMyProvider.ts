import { useQuery } from '@tanstack/react-query';
import { fetchMyProvider } from '../api';
import { useAppStore } from '../store/app-store';

export const MY_PROVIDER_QUERY_KEY = ['me', 'provider'] as const;

/**
 * The provider owned by the signed-in user (null when they have none).
 * Gated on `user` so anonymous visitors never call `/providers/me` (which
 * would 401 and trigger a refresh attempt). Pass `enabled` to override the
 * gate, e.g. on a page already behind `RequireAuth`.
 */
export function useMyProvider(enabled?: boolean) {
  const user = useAppStore((s) => s.user);
  return useQuery(MY_PROVIDER_QUERY_KEY, fetchMyProvider, {
    enabled: enabled ?? !!user,
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}

export default useMyProvider;
