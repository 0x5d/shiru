import { useEffect, useRef, useState } from 'react'
import { loginWithGoogle } from '../api'
import { useAuth } from '../useAuth'

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID as string

export default function LoginPage() {
  const buttonRef = useRef<HTMLDivElement>(null)
  const { setUser } = useAuth()
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    script.async = true
    script.onload = () => {
      if (cancelled) return
      window.google?.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: async (response) => {
          if (cancelled) return
          try {
            const user = await loginWithGoogle(response.credential)
            if (!cancelled) setUser(user)
          } catch {
            if (!cancelled) setError('Login failed. Please try again.')
          }
        },
      })
      if (buttonRef.current) {
        window.google?.accounts.id.renderButton(buttonRef.current, {
          type: 'standard',
          theme: 'outline',
          size: 'large',
          text: 'signin_with',
          shape: 'rectangular',
        })
      }
    }
    document.head.appendChild(script)
    return () => {
      cancelled = true
      script.remove()
    }
  }, [setUser])

  return (
    <div className="login-page">
      <div className="login-card">
        <h1 className="login-title">知る</h1>
        <p className="login-subtitle">Sign in to continue</p>
        {error && <p className="login-error">{error}</p>}
        <div ref={buttonRef} className="login-button" />
      </div>
    </div>
  )
}
