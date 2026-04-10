import { useState } from 'react'

const backend = (process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:8080')

export default function Register() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(null)

  async function onSubmit(e) {
    e.preventDefault()
    setError(null)
    setSuccess(null)
    try {
      const res = await fetch(`${backend}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
      if (!res.ok) throw new Error('Registration failed')
      const data = await res.json()
      setSuccess('Account created. You can login now.')
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="container">
      <h1>Register</h1>
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
        {success && <div style={{color: 'green'}}>{success}</div>}
        <button className="btn" type="submit" style={{marginTop: '1rem'}}>Register</button>
      </form>
    </div>
  )
}
