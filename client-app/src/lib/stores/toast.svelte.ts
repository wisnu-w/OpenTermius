// Clean error formatting & global toast notification store (Svelte 5 Runes)

export type ToastKind = 'error' | 'success' | 'info';

export interface ToastItem {
  id: number;
  kind: ToastKind;
  message: string;
}

const ERROR_MAP: Array<[RegExp, string]> = [
  [/failed to fetch|networkerror|load failed|network request failed/i, 'Cannot reach server. Check your internet connection or Gateway URL.'],
  [/econnrefused|connection refused/i, 'Connection refused. Make sure the target service is running.'],
  [/etimedout|timed out|timeout/i, 'Request timed out. Please try again.'],
  [/401|unauthorized|invalid token|expired/i, 'Session expired. Please sign in again.'],
  [/403|forbidden/i, 'You do not have permission to perform this action.'],
  [/404|not found/i, 'Requested item was not found.'],
  [/5\d\d|internal server error/i, 'Server encountered an error. Please try again shortly.']
];

export function formatErrorMessage(err: unknown, fallback = 'Something went wrong. Please try again.'): string {
  const raw =
    err instanceof Error
      ? err.message
      : typeof err === 'string'
        ? err
        : '';

  const cleaned = raw
    .replace(/^(Error|TypeError|SyntaxError):\s*/i, '')
    .replace(/\s+/g, ' ')
    .trim();

  if (!cleaned) return fallback;

  for (const [pattern, friendly] of ERROR_MAP) {
    if (pattern.test(cleaned)) return friendly;
  }

  // Strip raw JSON payloads if backend returned raw JSON string
  if (cleaned.startsWith('{') && cleaned.endsWith('}')) {
    try {
      const parsed = JSON.parse(cleaned) as { error?: string; message?: string };
      if (parsed.error) return parsed.error;
      if (parsed.message) return parsed.message;
    } catch {
      return fallback;
    }
  }

  return cleaned.length > 180 ? `${cleaned.slice(0, 177)}...` : cleaned;
}

class ToastStore {
  items = $state<ToastItem[]>([]);
  private nextId = 1;

  show(kind: ToastKind, message: string, durationMs?: number) {
    const id = this.nextId++;
    this.items = [...this.items.slice(-2), { id, kind, message }];
    const ms = durationMs ?? (kind === 'error' ? 5500 : 3500);
    setTimeout(() => this.dismiss(id), ms);
  }

  error(err: unknown, fallback?: string) {
    this.show('error', formatErrorMessage(err, fallback));
  }

  success(message: string) {
    this.show('success', message);
  }

  info(message: string) {
    this.show('info', message);
  }

  dismiss(id: number) {
    this.items = this.items.filter((t) => t.id !== id);
  }
}

export const toast = new ToastStore();
