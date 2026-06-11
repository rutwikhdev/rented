import { useState, useCallback, type ReactNode } from "react"
import { api } from "@/lib/api"
import { AuthContext } from "@/contexts/AuthContext"
import { getStoredSession, saveSession, clearSession } from "@/lib/session"

export function AuthProvider({ children }: { children: ReactNode }) {
  const [currentUser, setCurrentUser] = useState(() => getStoredSession())

  const login = useCallback(async (email: string, password: string) => {
    const session = await api.login(email, password)
    setCurrentUser(saveSession(session))
  }, [])

  const signup = useCallback(async (name: string, email: string, password: string, type: string) => {
    const session = await api.signup(name, email, password, type)
    setCurrentUser(saveSession(session))
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.logout()
    } finally {
      clearSession()
      setCurrentUser(null)
    }
  }, [])

  return (
    <AuthContext.Provider value={{ user: currentUser, loading: false, login, signup, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
