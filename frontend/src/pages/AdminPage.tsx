import PageTransition from '../components/PageTransition';
import { useState } from 'react';
import Spinner from '../components/Spinner';
import useUsers, { usePromoteUser } from '../hooks/useUsers';
import { adminListAudit } from '../api';
import { useQuery } from '@tanstack/react-query';
import Modal from '../components/Modal';
import toastLib from '../lib/toast';

export default function AdminPage() {
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const { data: users = [], isLoading: usersLoading } = useUsers(page, query);
  const { data: audit = [] } = useQuery<any[]>(['admin', 'audit'], async () => {
    const res = await adminListAudit();
    return res.audit ?? [];
  });

  const promoteMutation = usePromoteUser();
  const [confirmEmail, setConfirmEmail] = useState<string | null>(null);

  return (
    <PageTransition>
      <section className="mx-auto max-w-5xl px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Admin Dashboard</h2>
        {(usersLoading) ? (
          <div className="flex items-center justify-center"><Spinner /></div>
        ) : (
          <div className="grid gap-8 md:grid-cols-2">
            <div>
              <h3 className="mb-4 text-lg font-medium">Users</h3>
              <div className="mb-3 flex items-center justify-between gap-3">
                <input value={query} onChange={(e) => { setQuery(e.target.value); setPage(1); }} placeholder="Search by email or name" className="w-full rounded border px-3 py-2" />
              </div>
              <ul className="space-y-2">
                {users
                  .filter((u: any) => {
                    if (!query) return true;
                    const q = query.toLowerCase();
                    return (u.email || '').toLowerCase().includes(q) || (u.name || '').toLowerCase().includes(q);
                  })
                  .slice((page - 1) * pageSize, page * pageSize)
                  .map((u: any) => (
                  <li key={u.id} className="rounded border p-3">
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-medium">{u.email}</div>
                        <div className="text-sm text-muted-foreground">{u.name}</div>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="text-sm font-semibold">{u.role}</div>
                        {u.role !== 'admin' && (
                          <button
                            className="rounded bg-primary/10 px-3 py-1 text-sm disabled:opacity-50"
                            disabled={promoteMutation.isLoading}
                            onClick={async () => {
                              setConfirmEmail(u.email);
                            }}
                          >{promoteMutation.isLoading ? 'Promoting...' : 'Promote'}</button>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
              <div className="mt-3 flex items-center justify-between">
                <div className="text-sm text-muted-foreground">Showing page {page}</div>
                <div className="flex items-center gap-2">
                  <button disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))} className="rounded border px-2 py-1">Prev</button>
                  {(() => {
                    const q = query.toLowerCase();
                    const filteredCount = (users as any[]).filter((uu: any) => {
                      if (!query) return true;
                      return (uu.email || '').toLowerCase().includes(q) || (uu.name || '').toLowerCase().includes(q);
                    }).length;
                    return (
                      <button disabled={filteredCount <= page * pageSize} onClick={() => setPage((p) => p + 1)} className="rounded border px-2 py-1">Next</button>
                    );
                  })()}
                </div>
              </div>
            </div>
            <div>
              <h3 className="mb-4 text-lg font-medium">Audit</h3>
              <ul className="space-y-2">
                {(audit as any[]).map((a: any) => (
                  <li key={a.id} className="rounded border p-3">
                    <div className="text-sm font-medium">{a.action}</div>
                    <div className="text-xs text-muted-foreground">{a.created_at} — {a.target}</div>
                    <div className="text-sm">{a.details}</div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </section>
      <Modal open={!!confirmEmail} onClose={() => setConfirmEmail(null)} title="Confirm promotion">
        <p>Promote <strong>{confirmEmail}</strong> to admin?</p>
        <div className="mt-4 flex justify-end gap-2">
          <button className="rounded border px-3 py-1" onClick={() => setConfirmEmail(null)}>Cancel</button>
                <button
            className="rounded bg-primary px-3 py-1 text-white"
            onClick={async () => {
              try {
                if (!confirmEmail) return;
                await promoteMutation.mutateAsync(confirmEmail);
                toastLib.success(`${confirmEmail} promoted`);
              } catch (e) {
                toastLib.error('Promote failed');
              } finally {
                setConfirmEmail(null);
              }
            }}
          >{promoteMutation.isLoading ? 'Promoting...' : 'Confirm'}</button>
        </div>
      </Modal>
    </PageTransition>
  );
}
