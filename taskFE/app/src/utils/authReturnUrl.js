import { PostLoginReturnUrl } from '../domain/auth/value_objects/post_login_return_url_value_object.js'

export const POST_LOGIN_REDIRECT_STORAGE_KEY = 'postLoginRedirect'

export function savePostLoginRedirect(rawPath) {
  const normalized = PostLoginReturnUrl.normalize(rawPath)
  if (!normalized) return false
  try {
    localStorage.setItem(POST_LOGIN_REDIRECT_STORAGE_KEY, normalized)
    return true
  } catch {
    return false
  }
}
