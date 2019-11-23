import axios from 'axios';
import Cookies from 'js-cookie';

const TokenKey = "token"
const RefreshTokenKey = "RefreshToken"

export const API = axios.create({
  baseURL: "/api",
})

export default API

API.interceptors.request.use(config => {
  const token = Cookies.get(TokenKey)
  if (token) {
    // existing cookie will be sent along with request
    return config
  }
  
  const refreshToken = window.localStorage.getItem(RefreshTokenKey)
  if (! refreshToken) {
    return config
  }

  config.headers.Authorization = refreshToken
  return config
})

API.interceptors.response.use(null, error => {
  if (error.response && error.response.status === 401 && window.location.pathname !== "/web/login") {
    const params = new URLSearchParams()
    params.set("redirectURI", window.location)
    window.location = "/web/login?" + params.toString()
  }
  return Promise.reject(error)
})

export async function signIn(password) {
  if (! password) {
    throw new Error("Password is required")
  }
  const res = await API.post('/authz', {Password: password})
  // cookie should already be set by response headers, but we'll fall back with this since development proxy does weird things
  if (! Cookies.get(TokenKey)) {
    Cookies.set(TokenKey, res.data.Token, { expires: new Date(res.data.TokenExpiration) })
  }
  window.localStorage.setItem(RefreshTokenKey, res.data.RefreshToken)
}
