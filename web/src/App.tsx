import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom'
import { AuthProvider } from './AuthContext'
import { useAuth } from './useAuth'
import LoginPage from './pages/LoginPage'
import HomePage from './pages/HomePage'
import SettingsPage from './pages/SettingsPage'
import StoriesPage from './pages/StoriesPage'
import StoryReaderPage from './pages/StoryReaderPage'

function AppContent() {
  const auth = useAuth()

  if (auth.status === 'loading') {
    return <div className="container"><div className="loading" /></div>
  }

  if (auth.status === 'unauthenticated') {
    return <LoginPage />
  }

  return (
    <>
      <nav className="nav">
        <NavLink to="/" className="nav-brand">知る</NavLink>
        <NavLink to="/" end className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}>Home</NavLink>
        <NavLink to="/stories" className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}>Stories</NavLink>
        <NavLink to="/settings" className={({ isActive }) => `nav-link${isActive ? ' active' : ''}`}>Settings</NavLink>
        <button className="nav-logout" onClick={auth.logout}>Logout</button>
      </nav>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/stories" element={<StoriesPage />} />
        <Route path="/stories/:storyID" element={<StoryReaderPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Routes>
    </>
  )
}

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
