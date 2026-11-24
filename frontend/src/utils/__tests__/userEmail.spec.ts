import { describe, expect, it } from 'vitest'
import { shortUserEmail } from '../userEmail'
describe('user email presentation', () => {
  it('shortens Feishu placeholders without modifying real addresses', () => {
    expect(shortUserEmail('a'.repeat(64)+'@feishu-connect.invalid')).toBe('aaaaaaaa…@feishu')
    expect(shortUserEmail('staff@company.com')).toBe('staff@company.com')
  })
})
