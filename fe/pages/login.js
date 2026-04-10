import { useState } from 'react'

const backend = (process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080')

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)

  async function onSubmit(e) {
    e.preventDefault()
    setError(null)
    try {
      const res = await fetch(`${backend}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      if (!res.ok) throw new Error('Login failed')
      const data = await res.json()
      if (data.access_token) {
        localStorage.setItem('access_token', data.access_token)
        if (data.refresh_token) localStorage.setItem('refresh_token', data.refresh_token)
        window.location.href = '/'
      }
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="container">
      <h1>Login</h1>
      <form onSubmit={onSubmit} className="card" style={{maxWidth: 400}}>
        <div>
          <label>Email</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} required />
        </div>
        <div>
          <label>Password</label>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} required />
        </div>
        {error && <div style={{color: 'red'}}>{error}</div>}
        <button className="btn" type="submit" style={{marginTop: '1rem'}}>Login</button>
      </form>
    </div>
  )
}
