import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Inbox",
  description: "Send a message and files to the host, and get a reply.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <div className="mx-auto flex min-h-screen max-w-3xl flex-col px-4 py-8">
          <header className="mb-8 flex items-center justify-between">
            <Link href="/" className="text-lg font-semibold tracking-tight">
              Inbox
            </Link>
            <Link href="/inbox" className="text-sm text-slate-500 hover:underline">
              Host dashboard
            </Link>
          </header>
          <main className="flex-1">{children}</main>
          <footer className="mt-12 text-xs text-slate-400">
            Prototype v0.1
          </footer>
        </div>
      </body>
    </html>
  );
}
