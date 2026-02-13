import { Navigate } from "react-router-dom"

import { useAdminSessionState } from "@/hooks/auth/use-admin-session"

interface AuthGuardProps {
  children: React.ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  const { isAuthed } = useAdminSessionState()

  if (!isAuthed) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
