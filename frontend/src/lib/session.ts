import { type SessionData } from "@/lib/api"

interface User {
  id: string
  name: string
  email: string
  type: string
}

function sessionToUser(session: SessionData): User {
  return {
    id: session.user_id,
    name: session.user_name,
    email: session.user_email,
    type: session.user_type,
  }
}

export function getStoredSession(): User | null {
  const raw = localStorage.getItem("rented_session")
  if (!raw) return null
  try {
    const session: SessionData = JSON.parse(raw)
    return sessionToUser(session)
  } catch {
    return null
  }
}

export function saveSession(session: SessionData): User {
  localStorage.setItem("rented_session", JSON.stringify(session))
  return sessionToUser(session)
}

export function clearSession() {
  localStorage.removeItem("rented_session")
}
