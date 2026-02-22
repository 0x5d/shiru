import { useEffect, useRef } from 'react'
import { loginWithGoogle } from '../api'
import { useAuth } from '../useAuth'

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID as string

export default function LoginPage() {
  const buttonRef = useRef<HTMLDivElement>(null)
  const { setUser } = useAuth()

  useEffect(() => {
    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    script.async = true
    script.onload = () => {
      window.google?.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: async (response) => {
          const user = await loginWithGoogle(response.credential)
          setUser(user)
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
    return () => { script.remove() }
  }, [setUser])

  return (
    <div className="login-page">
      <div className="login-card">
        <h1 className="login-title">知る</h1>
        <p className="login-subtitle">Sign in to continue</p>
        <div ref={buttonRef} className="login-button" />
      </div>
    </div>
  )
}
