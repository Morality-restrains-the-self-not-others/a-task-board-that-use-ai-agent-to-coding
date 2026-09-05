// 密码加密工具，生成与 Django 兼容的 PBKDF2 密码哈希
// 说明：`crypto.subtle` 仅在「安全上下文」可用（HTTPS、localhost 等）。
// 公网 HTTP 下 `subtle` 不可用，须用 `@noble/hashes` 的 sha256 + pbkdf2 回退（勿用 npm 的 `create-hash` / `pbkdf2`，在 Vite 浏览器里会拖入 Node polyfill 导致 process/global 未定义）。

import { pbkdf2 } from '@noble/hashes/pbkdf2.js';
import { sha256 } from '@noble/hashes/sha2.js';

function hasWebCryptoSubtle() {
  try {
    return (
      typeof crypto !== 'undefined' &&
      crypto.subtle != null &&
      typeof crypto.subtle.digest === 'function' &&
      typeof crypto.subtle.importKey === 'function'
    );
  } catch {
    return false;
  }
}

// 生成随机盐
export async function generateSalt(length = 16) {
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode.apply(null, array));
}

/** PBKDF2-SHA256，输出与原先 Web Crypto 路径一致的 Base64（32 字节） */
function generatePBKDF2Fallback(password, salt, iterations = 180000, hashLength = 32) {
  const dk = pbkdf2(sha256, password, salt, { c: iterations, dkLen: hashLength });
  return btoa(String.fromCharCode.apply(null, [...dk]));
}

// 生成PBKDF2哈希
export async function generatePBKDF2(password, salt, iterations = 180000, hashLength = 32) {
  if (hasWebCryptoSubtle()) {
    const passwordBuffer = new TextEncoder().encode(password);
    const saltBuffer = new TextEncoder().encode(salt);

    const importedKey = await crypto.subtle.importKey(
      'raw',
      passwordBuffer,
      { name: 'PBKDF2' },
      false,
      ['deriveBits']
    );

    const derivedBits = await crypto.subtle.deriveBits(
      {
        name: 'PBKDF2',
        salt: saltBuffer,
        iterations: iterations,
        hash: { name: 'SHA-256' },
      },
      importedKey,
      hashLength * 8
    );

    return btoa(String.fromCharCode.apply(null, new Uint8Array(derivedBits)));
  }

  return generatePBKDF2Fallback(password, salt, iterations, hashLength);
}

// 生成Django兼容的密码哈希
export async function generateDjangoPasswordHash(password, username) {
  const iterations = 180000;

  const combinedString = password + username;
  let saltBytes16;

  if (hasWebCryptoSubtle()) {
    const combinedBuffer = new TextEncoder().encode(combinedString);
    const combinedHash = await crypto.subtle.digest('SHA-256', combinedBuffer);
    saltBytes16 = new Uint8Array(combinedHash).slice(0, 16);
  } else {
    const digest = sha256(new TextEncoder().encode(combinedString));
    saltBytes16 = digest.slice(0, 16);
  }

  const salt = btoa(String.fromCharCode.apply(null, [...saltBytes16]));
  const hash = await generatePBKDF2(password, salt, iterations, 32);

  return `pbkdf2_sha256$${iterations}$${salt}$${hash}`;
}

// 将函数挂载到window对象，以便其他模块可以使用
if (typeof window !== 'undefined') {
  window.PasswordHasher = {
    generateSalt,
    generatePBKDF2,
    generateDjangoPasswordHash,
  };

  window.generateDjangoPasswordHash = generateDjangoPasswordHash;
}
