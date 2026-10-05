"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";

import { api } from "@/lib/api";
import { formatDate } from "@/lib/format";
import type { ConversationDetail } from "@/lib/types";

export default function VisitorThreadPage() {
  const params = useParams<{ token: string }>();
  const token = params.token;
  const [detail, setDetail] = useState<ConversationDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .visitorThread(token)
      .then(setDetail)
      .catch(() => setError("This conversation link is invalid or has expired."));
  }, [token]);

  if (error) {
    return <p className="card text-sm text-slate-500">{error}</p>;
  }
  if (!detail) {
    return <p className="text-sm text-slate-500">Loading…</p>;
  }

  const { conversation, messages, attachments } = detail;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">
          {conversation.subject || "(no subject)"}
        </h1>
        <p className="text-sm text-slate-500">
          Conversation with the host
        </p>
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
                {m.direction === "outbound" ? "Host" : "You"}
              </span>
              <span>{formatDate(m.created_at)}</span>
            </div>
            <p className="whitespace-pre-wrap text-sm">{m.body}</p>
          </div>
        ))}
      </div>

      {attachments.length > 0 && (
        <div className="card">
          <h2 className="mb-2 text-sm font-semibold">Attachments</h2>
          <ul className="list-inside list-disc text-sm text-slate-600 dark:text-slate-400">
            {attachments.map((a) => (
              <li key={a.id}>{a.filename}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
