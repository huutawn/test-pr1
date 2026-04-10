import { useEffect, useState } from 'react'

function decodeJWT(token) {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload
  } catch {
    return null
  }
}

export default function Landing() {
  const [email, setEmail] = useState(null)
  useEffect(() => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null
    if (token) {
      const payload = decodeJWT(token)
      setEmail(payload?.email || null)
    }
  }, [])

  return (
    <div className="container">
      <h1>Welcome to Production Auth Demo</h1>
      {email ? (
        <div className="card" style={{marginTop: '1rem'}}>
          <p>You're logged in as {email}</p>
          <button className="btn" onClick={() => { localStorage.removeItem('access_token'); localStorage.removeItem('refresh_token'); window.location.reload(); }}>Logout</button>
        </div>
      ) : (
        <div className="grid" style={{marginTop: '1rem'}}>
          <div className="card">
            <h2>Landing Page</h2>
            <p>This is a production-ready authentication demo backed by a Go backend and a Next.js frontend.</p>
            <a href="/login"><button className="btn">Login</button></a>
          </div>
          <div className="card">
            <h2>Get Started</h2>
            <p>New here? Create an account to explore the auth flow.</p>
            <a href="/register"><button className="btn">Register</button></a>
          </div>
        </div>
      )}
    </div>
  )
}
