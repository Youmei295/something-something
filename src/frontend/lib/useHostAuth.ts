"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { api } from "./api";
import type { User } from "./types";

/** useHostAuth guards host pages, redirecting to /login when unauthenticated. */
export function useHostAuth() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    api
      .me()
      .then((u) => {
        if (active) setUser(u);
      })
      .catch(() => {
        router.replace("/login");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [router]);

  return { user, loading };
}
