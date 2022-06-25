import axios from 'axios'

const client = axios.create({
  baseURL: process.env.BASE_URL || 'http://localhost:8000',
})

export function setAuthHeader(token: string | null) {
  client.defaults.headers.Authorization = 'Bearer ' + token
}

export default client
