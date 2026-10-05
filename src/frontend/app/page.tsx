"use client";

import { useState } from "react";
import Link from "next/link";

import { ApiError, api, uploadFile } from "@/lib/api";
import type { SubmitResult } from "@/lib/types";

const MAX_FILE_BYTES = 5 * 1024 * 1024;

export default function SubmitPage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [result, setResult] = useState<SubmitResult | null>(null);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setFieldErrors({});

    if (file && file.size > MAX_FILE_BYTES) {
      setError("Attachment must be 5 MB or smaller.");
      return;
    }

    setSubmitting(true);
    try {
      const attachmentIds: string[] = [];
      if (file) {
        attachmentIds.push(await uploadFile(file));
      }
      const submitted = await api.submitMessage({
        name,
        email,
        subject,
        body,
        attachment_ids: attachmentIds,
      });
      setResult(submitted);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
        if (err.fields) setFieldErrors(err.fields);
      } else {
        setError("Something went wrong. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (result) {
    return (
      <div className="card space-y-4">
        <h1 className="text-xl font-semibold">Message sent</h1>
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Your message has been delivered to the host. Keep this link to read any
          reply.
        </p>
        <Link
          href={`/t/${result.visitor_token}`}
          className="btn inline-flex"
        >
          View conversation
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Send a message</h1>
        <p className="mt-1 text-sm text-slate-500">
          Questions, reports, or files — the host will reply by email.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="card space-y-4" noValidate>
        {error && (
          <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">
            {error}
          </p>
        )}

        <div>
          <label className="label" htmlFor="name">
            Your name
          </label>
          <input
            id="name"
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
          {fieldErrors.name && <FieldError message={fieldErrors.name} />}
        </div>

        <div>
          <label className="label" htmlFor="email">
            Your email
          </label>
          <input
            id="email"
            type="email"
            className="input"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
          {fieldErrors.email && <FieldError message={fieldErrors.email} />}
        </div>

        <div>
          <label className="label" htmlFor="subject">
            Subject
          </label>
          <input
            id="subject"
            className="input"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            required
          />
          {fieldErrors.subject && <FieldError message={fieldErrors.subject} />}
        </div>

        <div>
          <label className="label" htmlFor="body">
            Message
          </label>
          <textarea
            id="body"
            className="input min-h-32"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            required
          />
          {fieldErrors.body && <FieldError message={fieldErrors.body} />}
        </div>

        <div>
          <label className="label" htmlFor="file">
            Attachment (optional, max 5 MB)
          </label>
          <input
            id="file"
            type="file"
            className="block w-full text-sm text-slate-500 file:mr-3 file:rounded-lg file:border-0 file:bg-slate-900 file:px-3 file:py-2 file:text-sm file:text-white dark:file:bg-white dark:file:text-slate-900"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            accept="image/png,image/jpeg,image/gif,image/webp,application/pdf"
          />
        </div>

        <button className="btn w-full" disabled={submitting}>
          {submitting ? "Sending…" : "Send message"}
        </button>
      </form>
    </div>
  );
}

function FieldError({ message }: { message: string }) {
  return <p className="mt-1 text-xs text-red-600">{message}</p>;
}
