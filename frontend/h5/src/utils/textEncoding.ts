const hasCjk = /[\u3400-\u9fff]/;
const hasLatin1HighBytes = /[\u0080-\u00ff]/;

export function repairUtf8Mojibake(text?: string | null): string {
  if (!text || hasCjk.test(text) || !hasLatin1HighBytes.test(text)) {
    return text || '';
  }

  try {
    const escapedBytes = Array.from(text)
      .map((char) => `%${char.charCodeAt(0).toString(16).padStart(2, '0')}`)
      .join('');
    const repaired = decodeURIComponent(escapedBytes);
    return hasCjk.test(repaired) ? repaired : text;
  } catch {
    return text;
  }
}
