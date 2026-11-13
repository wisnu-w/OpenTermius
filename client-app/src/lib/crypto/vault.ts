// OpenTermius Client-Side WebCrypto E2EE Adapter

export async function deriveKeyFromPassword(password: string, salt: Uint8Array): Promise<CryptoKey> {
  const enc = new TextEncoder();
  const keyMaterial = await window.crypto.subtle.importKey(
    'raw',
    enc.encode(password),
    'PBKDF2',
    false,
    ['deriveBits', 'deriveKey']
  );

  return window.crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: 100000,
      hash: 'SHA-256'
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    true,
    ['encrypt', 'decrypt']
  );
}

export async function encryptVaultItem(
  payload: unknown,
  vaultKey: CryptoKey
): Promise<{ ciphertext: string; nonce: string }> {
  const enc = new TextEncoder();
  const jsonBytes = enc.encode(JSON.stringify(payload));

  // Generate 96-bit (12-byte) initialization vector
  const iv = window.crypto.getRandomValues(new Uint8Array(12));

  const encryptedBuffer = await window.crypto.subtle.encrypt(
    {
      name: 'AES-GCM',
      iv
    },
    vaultKey,
    jsonBytes
  );

  return {
    ciphertext: bufferToBase64(new Uint8Array(encryptedBuffer)),
    nonce: bufferToBase64(iv)
  };
}

export async function decryptVaultItem<T>(
  ciphertextBase64: string,
  nonceBase64: string,
  vaultKey: CryptoKey
): Promise<T> {
  const iv = base64ToBuffer(nonceBase64);
  const encryptedBytes = base64ToBuffer(ciphertextBase64);

  const decryptedBuffer = await window.crypto.subtle.decrypt(
    {
      name: 'AES-GCM',
      iv
    },
    vaultKey,
    encryptedBytes
  );

  const dec = new TextDecoder();
  return JSON.parse(dec.decode(decryptedBuffer)) as T;
}

function bufferToBase64(buffer: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < buffer.byteLength; i++) {
    binary += String.fromCharCode(buffer[i]);
  }
  return btoa(binary);
}

function base64ToBuffer(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
