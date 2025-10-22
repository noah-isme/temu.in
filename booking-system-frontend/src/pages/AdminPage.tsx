import PageTransition from '../components/PageTransition';
import { useEffect, useState } from 'react';
import { adminListUsers, adminListAudit, adminPromoteUser } from '../api';
import Spinner from '../components/Spinner';

export default function AdminPage() {
  const [users, setUsers] = useState<any[]>([]);
  const [audit, setAudit] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const pageSize = 10;
  const [promoting, setPromoting] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    Promise.all([adminListUsers(), adminListAudit()])
      .then(([u, a]) => {
        if (!mounted) return;
        setUsers(u.users ?? []);
        setAudit(a.audit ?? []);
      })
      .catch(() => {})
      .finally(() => mounted && setLoading(false));
    return () => { mounted = false };
  }, []);

  return (
    <PageTransition>
      <section className="mx-auto max-w-5xl px-6 py-12">
        <h2 className="mb-6 text-2xl font-semibold">Admin Dashboard</h2>
        {loading ? (
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
                  .filter((u) => {
                    if (!query) return true;
                    const q = query.toLowerCase();
                    return (u.email || '').toLowerCase().includes(q) || (u.name || '').toLowerCase().includes(q);
                  })
                  .slice((page - 1) * pageSize, page * pageSize)
                  .map((u) => (
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
                            disabled={promoting !== null}
                            onClick={async () => {
                              const ok = window.confirm(`Promote ${u.email} to admin?`);
                              if (!ok) return;
                              try {
                                setPromoting(u.email);
                                await adminPromoteUser(u.email);
                                // refresh lists
                                setLoading(true);
                                const [uRes, aRes] = await Promise.all([adminListUsers(), adminListAudit()]);
                                setUsers(uRes.users ?? []);
                                setAudit(aRes.audit ?? []);
                              } catch (e: any) {
                                console.error(e);
                                window.alert('Promote failed');
                              } finally {
                                setLoading(false);
                                setPromoting(null);
                              }
                            }}
                          >{promoting === u.email ? 'Promoting...' : 'Promote'}</button>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
              <div className="mt-3 flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                  Showing page {page}
                </div>
                <div className="flex items-center gap-2">
                  <button disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))} className="rounded border px-2 py-1">Prev</button>
                  <button disabled={(users.filter((u) => { const q = query.toLowerCase(); return !query || (u.email||'').toLowerCase().includes(q) || (u.name||'').toLowerCase().includes(q); }).length <= page * pageSize)} onClick={() => setPage((p) => p + 1)} className="rounded border px-2 py-1">Next</button>
                </div>
              </div>
            </div>
            <div>
              <h3 className="mb-4 text-lg font-medium">Audit</h3>
              <ul className="space-y-2">
                {audit.map((a) => (
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
    </PageTransition>
  );
}
