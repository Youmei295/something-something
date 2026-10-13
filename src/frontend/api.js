// Shared API client and small DOM helpers for every page.
//
// Backend origin configuration:
//   - Local dev (served on :3000) talks to http://localhost:8080 automatically.
//   - When deploying elsewhere, set window.API_BASE or PROD_API_BASE below.
var PROD_API_BASE = ""; // e.g. "https://api.example.com"

var API_BASE = (
  (window.API_BASE || PROD_API_BASE || "").replace(/\/$/, "") ||
  (location.port === "3000" ? "http://localhost:8080" : "")
);

/** ApiError mirrors the backend error shape: { error, fields? }. */
function ApiError(status, message, fields) {
  this.name = "ApiError";
  this.status = status;
  this.message = message;
  this.fields = fields || null;
}
ApiError.prototype = Object.create(Error.prototype);

async function request(path, init) {
  init = init || {};
  var headers = Object.assign({ "Content-Type": "application/json" }, init.headers || {});
  var res = await fetch(API_BASE + path, Object.assign({}, init, {
    credentials: "include",
    headers: headers,
  }));
  if (res.status === 204) return undefined;
  var text = await res.text();
  var data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new ApiError(res.status, (data && data.error) || res.statusText, data && data.fields);
  }
  return data;
}

var api = {
  // Auth
  me: function () { return request("/api/v1/auth/me"); },
  login: function (email, password) {
    return request("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ email: email, password: password }) });
  },
  logout: function () { return request("/api/v1/auth/logout", { method: "POST" }); },

  // Visitor
  submitMessage: function (payload) {
    return request("/api/v1/messages", { method: "POST", body: JSON.stringify(payload) });
  },
  visitorThread: function (token) {
    return request("/api/v1/visitor/threads/" + encodeURIComponent(token));
  },

  // Conversations (host)
  listConversations: function (params) {
    params = params || {};
    var q = new URLSearchParams();
    if (params.status) q.set("status", params.status);
    if (params.q) q.set("q", params.q);
    if (params.limit) q.set("limit", String(params.limit));
    var suffix = q.toString() ? "?" + q.toString() : "";
    return request("/api/v1/conversations" + suffix);
  },
  getConversation: function (id) { return request("/api/v1/conversations/" + id); },
  reply: function (id, body, attachmentIds) {
    return request("/api/v1/conversations/" + id + "/replies", {
      method: "POST",
      body: JSON.stringify({ body: body, attachment_ids: attachmentIds || [] }),
    });
  },
  updateConversation: function (id, patch) {
    return request("/api/v1/conversations/" + id, { method: "PATCH", body: JSON.stringify(patch) });
  },
  deleteConversation: function (id) {
    return request("/api/v1/conversations/" + id, { method: "DELETE" });
  },

  // Attachments
  requestUpload: function (filename, contentType, size) {
    return request("/api/v1/attachments/upload-url", {
      method: "POST",
      body: JSON.stringify({ filename: filename, content_type: contentType, size: size }),
    });
  },
  completeUpload: function (id) {
    return request("/api/v1/attachments/" + id + "/complete", { method: "POST" });
  },
  downloadUrl: function (id) { return request("/api/v1/attachments/" + id + "/download"); },
};

/** uploadFile runs the presigned flow and resolves to the attachment id. */
async function uploadFile(file) {
  var contentType = file.type || "application/octet-stream";
  var ticket = await api.requestUpload(file.name, contentType, file.size);
  var put = await fetch(ticket.upload_url, { method: "PUT", headers: { "Content-Type": contentType }, body: file });
  if (!put.ok) throw new ApiError(put.status, "upload failed");
  await api.completeUpload(ticket.attachment_id);
  return ticket.attachment_id;
}

/* ---- helpers ---- */

function $(sel, root) { return (root || document).querySelector(sel); }

/** h("div", { class: "x", text: "hi" }, childNode, ...) builds an element safely. */
function h(tag, props, ...children) {
  var e = document.createElement(tag);
  Object.keys(props || {}).forEach(function (k) {
    var v = props[k];
    if (k === "class") e.className = v;
    else if (k === "text") e.textContent = v;
    else if (k.indexOf("on") === 0) e.addEventListener(k.slice(2).toLowerCase(), v);
    else if (v != null) e.setAttribute(k, v);
  });
  children.forEach(function (c) { if (c != null) e.append(c); });
  return e;
}

function formatDate(value) {
  var d = new Date(value);
  return isNaN(d) ? "" : d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function formatBytes(n) {
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
  return (n / (1024 * 1024)).toFixed(1) + " MB";
}

function showAlert(node, message) {
  node.textContent = message;
  node.classList.remove("hidden");
}

function hideAlert(node) {
  node.textContent = "";
  node.classList.add("hidden");
}

function clearFieldErrors(form) {
  form.querySelectorAll(".field-error").forEach(function (e) { e.textContent = ""; });
}

function showFieldErrors(form, fields) {
  Object.keys(fields || {}).forEach(function (name) {
    var node = form.querySelector('[data-error="' + name + '"]');
    if (node) node.textContent = fields[name];
  });
}

/** requireAuth guards host pages, redirecting to login when unauthenticated. */
async function requireAuth() {
  try {
    return await api.me();
  } catch (e) {
    location.replace("login.html");
    return null;
  }
}

async function logout() {
  try { await api.logout(); } catch (e) { /* ignore */ }
  location.replace("login.html");
}
