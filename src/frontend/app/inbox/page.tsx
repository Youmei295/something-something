"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { api } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useHostAuth } from "@/lib/useHostAuth";
import type { Conversation } from "@/lib/types";

export default function InboxPage() {
  const router = useRouter();
  const { user, loading } = useHostAuth();
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState("open");
  const [query, setQuery] = useState("");
  const [loadingList, setLoadingList] = useState(true);

  const load = useCallback(async () => {
    setLoadingList(true);
    try {
      const data = await api.listConversations({ status, q: query, limit: 50 });
      setConversations(data.items);
      setTotal(data.total);
    } finally {
      setLoadingList(false);
    }
  }, [status, query]);

  useEffect(() => {
    if (user) void load();
  }, [user, load]);

  async function handleLogout() {
    await api.logout().catch(() => undefined);
    router.replace("/login");
  }

  if (loading) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Inbox</h1>
        <div className="flex items-center gap-3 text-sm">
          <span className="text-slate-500">{user?.email}</span>
          <button className="btn-secondary" onClick={handleLogout}>
            Sign out
          </button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <input
          className="input max-w-xs"
          placeholder="Search…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <select
          className="input max-w-[10rem]"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="open">Open</option>
          <option value="archived">Archived</option>
          <option value="all">All</option>
        </select>
        <button className="btn-secondary" onClick={() => void load()}>
          Refresh
        </button>
        <span className="text-sm text-slate-500">{total} total</span>
      </div>

      <div className="divide-y divide-slate-200 overflow-hidden rounded-xl border border-slate-200 bg-white dark:divide-slate-800 dark:border-slate-800 dark:bg-slate-900">
        {loadingList ? (
          <p className="p-6 text-sm text-slate-500">Loading…</p>
        ) : conversations.length === 0 ? (
          <p className="p-6 text-sm text-slate-500">No conversations.</p>
        ) : (
          conversations.map((c) => (
            <Link
              key={c.id}
              href={`/inbox/${c.id}`}
              className="flex items-start justify-between gap-4 p-4 transition hover:bg-slate-50 dark:hover:bg-slate-800"
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  {c.unread && (
                    <span className="h-2 w-2 rounded-full bg-blue-500" />
                  )}
                  <span className="truncate font-medium">
                    {c.subject || "(no subject)"}
                  </span>
                </div>
                <p className="truncate text-sm text-slate-500">
                  {c.visitor_name} · {c.visitor_email}
                </p>
              </div>
              <span className="shrink-0 text-xs text-slate-400">
                {formatDate(c.last_activity_at)}
              </span>
            </Link>
          ))
        )}
      </div>
    </div>
  );
}
