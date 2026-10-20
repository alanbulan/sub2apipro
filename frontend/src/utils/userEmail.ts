// Presentation only: keep the stable login identifier in the database and in
// copy/title controls, even when Feishu cannot supply a real employee email.
export function shortUserEmail(email: string): string {
  if (email.endsWith('@feishu-connect.invalid')) return `${email.slice(0, 8)}…@feishu`
  if (email.length <= 32) return email
  const at = email.lastIndexOf('@')
  return at > 12 ? `${email.slice(0, 10)}…${email.slice(at)}` : `${email.slice(0, 28)}…`
}
