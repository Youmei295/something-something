import type {
  Attachment,
  ConversationDetail,
  ListConversations,
  LoginResult,
  Message,
  SubmitResult,
  UploadTicket,
  User,
} from "./types";

const API_BASE = (
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"
).replace(/\/$/, "");

/** ApiError carries the backend's error shape so UIs can show field errors. */
export class ApiError extends Error {
  status: number;
  fields?: Record<string, string>;

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fields = fields;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  const data = text ? JSON.parse(text) : null;

  if (!res.ok) {
    throw new ApiError(
      res.status,
      data?.error ?? res.statusText,
      data?.fields,
    );
  }
  return data as T;
}

export const api = {
  // Auth
  me: () => request<User>("/api/v1/auth/me"),
  login: (email: string, password: string) =>
    request<LoginResult>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  logout: () =>
    request<void>("/api/v1/auth/logout", { method: "POST" }),

  // Visitor
  submitMessage: (payload: {
    name: string;
    email: string;
    subject: string;
    body: string;
    attachment_ids?: string[];
  }) =>
    request<SubmitResult>("/api/v1/messages", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  visitorThread: (token: string) =>
    request<ConversationDetail>(
      `/api/v1/visitor/threads/${encodeURIComponent(token)}`,
    ),

  // Conversations (host)
  listConversations: (params: {
    status?: string;
    q?: string;
    limit?: number;
    offset?: number;
  } = {}) => {
    const query = new URLSearchParams();
    if (params.status) query.set("status", params.status);
    if (params.q) query.set("q", params.q);
    if (params.limit) query.set("limit", String(params.limit));
    if (params.offset) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query}` : "";
    return request<ListConversations>(`/api/v1/conversations${suffix}`);
  },
  getConversation: (id: string) =>
    request<ConversationDetail>(`/api/v1/conversations/${id}`),
  reply: (id: string, body: string, attachmentIds: string[] = []) =>
    request<Message>(`/api/v1/conversations/${id}/replies`, {
      method: "POST",
      body: JSON.stringify({ body, attachment_ids: attachmentIds }),
    }),
  updateConversation: (
    id: string,
    patch: { read?: boolean; archived?: boolean },
  ) =>
    request<ConversationDetail>(`/api/v1/conversations/${id}`, {
      method: "PATCH",
      body: JSON.stringify(patch),
    }),
  deleteConversation: (id: string) =>
    request<void>(`/api/v1/conversations/${id}`, { method: "DELETE" }),

  // Attachments
  requestUpload: (filename: string, contentType: string, size: number) =>
    request<UploadTicket>("/api/v1/attachments/upload-url", {
      method: "POST",
      body: JSON.stringify({
        filename,
        content_type: contentType,
        size,
      }),
    }),
  completeUpload: (id: string) =>
    request<Attachment>(`/api/v1/attachments/${id}/complete`, {
      method: "POST",
    }),
  downloadUrl: (id: string) =>
    request<{ url: string }>(`/api/v1/attachments/${id}/download`),
};

/**
 * uploadFile performs the full presigned flow: request a URL, PUT the bytes
 * straight to storage, then mark the upload complete. Returns the attachment id.
 */
export async function uploadFile(file: File): Promise<string> {
  const contentType = file.type || "application/octet-stream";
  const ticket = await api.requestUpload(file.name, contentType, file.size);

  const put = await fetch(ticket.upload_url, {
    method: "PUT",
    headers: { "Content-Type": contentType },
    body: file,
  });
  if (!put.ok) {
    throw new ApiError(put.status, "upload failed");
  }

  await api.completeUpload(ticket.attachment_id);
  return ticket.attachment_id;
}
