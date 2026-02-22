interface CredentialResponse {
  credential: string
}

interface GsiButtonConfiguration {
  type: 'standard' | 'icon'
  theme?: 'outline' | 'filled_blue' | 'filled_black'
  size?: 'large' | 'medium' | 'small'
  text?: 'signin_with' | 'signup_with' | 'continue_with' | 'signin'
  shape?: 'rectangular' | 'pill' | 'circle' | 'square'
  width?: number
}

interface IdConfiguration {
  client_id: string
  callback: (response: CredentialResponse) => void
  auto_select?: boolean
}

interface Google {
  accounts: {
    id: {
      initialize: (config: IdConfiguration) => void
      renderButton: (parent: HTMLElement, config: GsiButtonConfiguration) => void
      revoke: (hint: string, callback?: () => void) => void
    }
  }
}

interface Window {
  google?: Google
}
