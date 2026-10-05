"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";

import { ApiError, api, uploadFile } from "@/lib/api";
import { formatBytes, formatDate } from "@/lib/format";
import { useHostAuth } from "@/lib/useHostAuth";
import type { ConversationDetail } from "@/lib/types";

export default function ConversationPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const router = useRouter();
  const { user, loading } = useHostAuth();

  const [detail, setDetail] = useState<ConversationDetail | null>(null);
  const [reply, setReply] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const data = await api.getConversation(id);
    setDetail(data);
    if (data.conversation.unread) {
      await api.updateConversation(id, { read: true }).catch(() => undefined);
    }
  }, [id]);

  useEffect(() => {
    if (user) void load().catch(() => setError("Conversation not found."));
  }, [user, load]);

  async function handleReply(event: React.FormEvent) {
    event.preventDefault();
    if (!reply.trim()) return;
    setSending(true);
    setError(null);
    try {
      const attachmentIds = file ? [await uploadFile(file)] : [];
      await api.reply(id, reply, attachmentIds);
      setReply("");
      setFile(null);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to send reply.");
    } finally {
      setSending(false);
    }
  }

  async function handleDownload(attachmentId: string) {
    const { url } = await api.downloadUrl(attachmentId);
    window.open(url, "_blank", "noopener,noreferrer");
  }

  async function handleArchive() {
    if (!detail) return;
    const archived = detail.conversation.status !== "archived";
    await api.updateConversation(id, { archived });
    router.push("/inbox");
  }

  async function handleDelete() {
    if (!confirm("Delete this conversation?")) return;
    await api.deleteConversation(id);
    router.push("/inbox");
  }

  if (loading || !detail) {
    return <p className="text-sm text-slate-500">{error ?? "Loading…"}</p>;
  }

  const { conversation, messages, attachments } = detail;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link href="/inbox" className="text-sm text-slate-500 hover:underline">
            ← Back to inbox
          </Link>
          <h1 className="mt-1 text-2xl font-semibold">
            {conversation.subject || "(no subject)"}
          </h1>
          <p className="text-sm text-slate-500">
            {conversation.visitor_name} · {conversation.visitor_email}
          </p>
        </div>
        <div className="flex gap-2">
          <button className="btn-secondary" onClick={handleArchive}>
            {conversation.status === "archived" ? "Restore" : "Archive"}
          </button>
          <button className="btn-secondary text-red-600" onClick={handleDelete}>
            Delete
          </button>
        </div>
      </div>

      <div className="space-y-3">
        {messages.map((m) => (
          <div
            key={m.id}
            className={`card ${
              m.direction === "outbound"
                ? "border-blue-200 bg-blue-50 dark:border-blue-900 dark:bg-blue-950/40"
                : ""
            }`}
          >
            <div className="mb-2 flex items-center justify-between text-xs text-slate-500">
              <span className="font-medium uppercase tracking-wide">
                {m.direction === "outbound" ? "Host" : conversation.visitor_name}
              </span>
              <span>{formatDate(m.created_at)}</span>
            </div>
            <p className="whitespace-pre-wrap text-sm">{m.body}</p>
          </div>
        ))}
      </div>

      {attachments.length > 0 && (
        <div className="card">
          <h2 className="mb-3 text-sm font-semibold">Attachments</h2>
          <ul className="space-y-2">
            {attachments.map((a) => (
              <li key={a.id} className="flex items-center justify-between gap-3">
                <div className="min-w-0 text-sm">
                  <p className="truncate">{a.filename}</p>
                  <p className="text-xs text-slate-500">
                    {formatBytes(a.size)} · {a.scan_status}
                  </p>
                </div>
                <button
                  className="btn-secondary"
                  onClick={() => handleDownload(a.id)}
                  disabled={a.scan_status !== "clean"}
                >
                  Download
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      <form onSubmit={handleReply} className="card space-y-3">
        <h2 className="text-sm font-semibold">Reply</h2>
        {error && (
          <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">
            {error}
          </p>
        )}
        <textarea
          className="input min-h-28"
          value={reply}
          onChange={(e) => setReply(e.target.value)}
          placeholder="Write your reply…"
          required
        />
        <input
          type="file"
          className="block w-full text-sm text-slate-500 file:mr-3 file:rounded-lg file:border-0 file:bg-slate-900 file:px-3 file:py-2 file:text-sm file:text-white dark:file:bg-white dark:file:text-slate-900"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          accept="image/png,image/jpeg,image/gif,image/webp,application/pdf"
        />
        <button className="btn" disabled={sending}>
          {sending ? "Sending…" : "Send reply"}
        </button>
      </form>
    </div>
  );
}
