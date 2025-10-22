import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { adminListUsers, adminPromoteUser } from '../api';

export function useUsers(page = 1, q = '') {
  return useQuery(['admin', 'users', { page, q }], async () => {
    const res = await adminListUsers();
    // backend currently returns full list; client-side filter/paginate if needed
    return res.users ?? [];
  }, {
    keepPreviousData: true,
    staleTime: 1000 * 10,
  });
}

export function usePromoteUser() {
  const qc = useQueryClient();
  return useMutation((email: string) => adminPromoteUser(email), {
    // optimistic update: mark role as admin in users cache
    onMutate: async (email: string) => {
      await qc.cancelQueries(['admin', 'users']);
      const previous = qc.getQueryData<any[]>(['admin', 'users', { page: 1, q: '' }]) ?? qc.getQueryData<any[]>(['admin', 'users']);
      if (previous) {
        qc.setQueryData(['admin', 'users', { page: 1, q: '' }], previous.map(u => u.email === email ? { ...u, role: 'admin' } : u));
        qc.setQueryData(['admin', 'users'], previous.map(u => u.email === email ? { ...u, role: 'admin' } : u));
      }
      return { previous };
    },
    onError: (err, variables, context: any) => {
      if (context?.previous) {
        qc.setQueryData(['admin', 'users', { page: 1, q: '' }], context.previous);
        qc.setQueryData(['admin', 'users'], context.previous);
      }
    },
    onSettled: () => {
      qc.invalidateQueries(['admin', 'users']);
      qc.invalidateQueries(['admin', 'audit']);
    }
  });
}

export default useUsers;
