export type User = {
  id: string;
  email: string;
  role: string;
};

export type Conversation = {
  id: string;
  subject: string;
  status: "open" | "archived";
  visitor_name: string;
  visitor_email: string;
  unread: boolean;
  created_at: string;
  last_activity_at: string;
};

export type Message = {
  id: string;
  direction: "inbound" | "outbound";
  body: string;
  created_at: string;
};

export type Attachment = {
  id: string;
  filename: string;
  content_type: string;
  size: number;
  scan_status: "pending" | "clean" | "infected" | "failed";
};

export type ConversationDetail = {
  conversation: Conversation;
  messages: Message[];
  attachments: Attachment[];
};

export type ListConversations = {
  items: Conversation[];
  total: number;
  limit: number;
  offset: number;
};

export type SubmitResult = {
  conversation_id: string;
  visitor_token: string;
  created_at: string;
};

export type UploadTicket = {
  attachment_id: string;
  upload_url: string;
  storage_key: string;
  expires_in: number;
};

export type LoginResult = {
  user: User;
  expires_at: string;
};
