import { Navigate } from "react-router-dom"

import { useUserSessionState } from "@/hooks/user/use-auth-hooks"

interface AuthGuardProps {
  children: React.ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  const { isAuthed } = useUserSessionState()

  if (!isAuthed) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
