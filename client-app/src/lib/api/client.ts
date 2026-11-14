// OpenTermius Production API Client & Transport Layer

export interface AuthResponse {
  user: {
    id: string;
    email: string;
    name: string;
    created_at: string;
  };
  token: string;
}

export interface ApiVault {
  id: string;
  owner_id: string;
  name: string;
  type: 'personal' | 'team';
  encrypted_key: string;
  created_at: string;
  updated_at: string;
}

export interface ApiVaultItem {
  id: string;
  vault_id: string;
  kind: 'host' | 'key' | 'identity' | 'snippet' | 'tunnel';
  label: string;
  ciphertext: string;
  nonce: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface ApiVaultMember {
  id: string;
  vault_id: string;
  user_id: string;
  role: 'owner' | 'admin' | 'member';
  created_at: string;
}

export interface ApiSftpFile {
  name: string;
  path: string;
  isDir: boolean;
  size: number;
  modTime: string;
  permissions: string;
}

export interface SftpHostConfig {
  address: string;
  port?: number;
  username: string;
  authType?: 'password' | 'key';
  password?: string;
  privateKey?: string;
  passphrase?: string;
}

type SyncQueueOp =
  | {
      op: 'create';
      vaultId: string;
      tempId: string;
      item: {
        kind: 'host' | 'key' | 'identity' | 'snippet' | 'tunnel';
        label: string;
        ciphertext: string;
        nonce: string;
      };
    }
  | {
      op: 'update';
      vaultId: string;
      itemId: string;
      item: {
        label: string;
        ciphertext: string;
        nonce: string;
      };
    }
  | {
      op: 'delete';
      vaultId: string;
      itemId: string;
    }
  | {
      op: 'move';
      sourceVaultId: string;
      itemId: string;
      targetVaultId: string;
    };

const GW_USER_CACHE_KEY = 'opentermius_gw_user_cache';
const GW_VAULTS_CACHE_KEY = 'opentermius_gw_vaults_cache';
const GW_ITEMS_CACHE_PREFIX = 'opentermius_gw_items_cache_';
const GW_SYNC_QUEUE_KEY = 'opentermius_gw_sync_queue';

class ApiClient {
  private token: string | null = null;
  private apiBase: string = 'http://localhost:8080/api/v1';
  private standalone: boolean = false;
  private isFlushing: boolean = false;
  private onSyncListeners: Set<(syncedCount: number) => void> = new Set();

  constructor() {
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('opentermius_jwt_token');
      this.apiBase = localStorage.getItem('opentermius_api_base') || 'http://localhost:8080/api/v1';
      this.standalone = localStorage.getItem('opentermius_mode') === 'standalone';
      window.addEventListener('online', () => {
        if (!this.standalone && this.token) {
          void this.flushSyncQueue();
        }
      });
    }
  }

  onSyncComplete(cb: (syncedCount: number) => void): () => void {
    this.onSyncListeners.add(cb);
    return () => this.onSyncListeners.delete(cb);
  }

  getPendingSyncCount(): number {
    return this.readSyncQueue().length;
  }

  private isNetworkError(err: unknown): boolean {
    if (typeof navigator !== 'undefined' && !navigator.onLine) return true;
    if (err instanceof TypeError) return true;
    const msg = err instanceof Error ? err.message : String(err);
    return /failed to fetch|networkerror|load failed|network request failed|econnrefused|timed out|timeout/i.test(msg);
  }

  private readJson<T>(key: string, fallback: T): T {
    if (typeof window === 'undefined') return fallback;
    try {
      const raw = localStorage.getItem(key);
      return raw ? (JSON.parse(raw) as T) : fallback;
    } catch {
      return fallback;
    }
  }

  private writeJson(key: string, value: unknown) {
    if (typeof window === 'undefined') return;
    localStorage.setItem(key, JSON.stringify(value));
  }

  private readSyncQueue(): SyncQueueOp[] {
    return this.readJson<SyncQueueOp[]>(GW_SYNC_QUEUE_KEY, []);
  }

  private writeSyncQueue(queue: SyncQueueOp[]) {
    this.writeJson(GW_SYNC_QUEUE_KEY, queue);
  }

  private readGatewayItemsCache(vaultId: string): ApiVaultItem[] {
    return this.readJson<ApiVaultItem[]>(`${GW_ITEMS_CACHE_PREFIX}${vaultId}`, []);
  }

  private writeGatewayItemsCache(vaultId: string, items: ApiVaultItem[]) {
    this.writeJson(`${GW_ITEMS_CACHE_PREFIX}${vaultId}`, items);
  }

  private enqueueOfflineOp(op: SyncQueueOp) {
    const queue = this.readSyncQueue();
    if (op.op === 'update' && op.itemId.startsWith('local-pending-')) {
      const createIdx = queue.findIndex((q) => q.op === 'create' && q.tempId === op.itemId);
      if (createIdx !== -1) {
        const existing = queue[createIdx] as Extract<SyncQueueOp, { op: 'create' }>;
        existing.item = { ...existing.item, ...op.item };
        this.writeSyncQueue(queue);
        return;
      }
    }
    if (op.op === 'delete' && op.itemId.startsWith('local-pending-')) {
      const filtered = queue.filter((q) => !(q.op === 'create' && q.tempId === op.itemId) && !('itemId' in q && q.itemId === op.itemId));
      this.writeSyncQueue(filtered);
      return;
    }
    if (op.op === 'move' && op.itemId.startsWith('local-pending-')) {
      const createIdx = queue.findIndex((q) => q.op === 'create' && q.tempId === op.itemId);
      if (createIdx !== -1) {
        const existing = queue[createIdx] as Extract<SyncQueueOp, { op: 'create' }>;
        existing.vaultId = op.targetVaultId;
        this.writeSyncQueue(queue);
        return;
      }
    }
    queue.push(op);
    this.writeSyncQueue(queue);
  }

  async flushSyncQueue(): Promise<number> {
    if (this.standalone || !this.token || this.isFlushing) return 0;
    const queue = this.readSyncQueue();
    if (queue.length === 0) return 0;

    this.isFlushing = true;
    let syncedCount = 0;
    const idMap = new Map<string, string>();
    const remaining: SyncQueueOp[] = [];

    try {
      for (let i = 0; i < queue.length; i++) {
        const op = queue[i];
        try {
          if (op.op === 'create') {
            const created = await this.request<ApiVaultItem>(`/vaults/${op.vaultId}/items`, {
              method: 'POST',
              body: JSON.stringify(op.item)
            });
            idMap.set(op.tempId, created.id);
            const cached = this.readGatewayItemsCache(op.vaultId);
            this.writeGatewayItemsCache(
              op.vaultId,
              cached.map((c) => (c.id === op.tempId ? created : c))
            );
            syncedCount++;
          } else if (op.op === 'update') {
            const targetId = idMap.get(op.itemId) ?? op.itemId;
            await this.request<ApiVaultItem>(`/vaults/${op.vaultId}/items/${targetId}`, {
              method: 'PUT',
              body: JSON.stringify(op.item)
            });
            syncedCount++;
          } else if (op.op === 'delete') {
            const targetId = idMap.get(op.itemId) ?? op.itemId;
            await this.request<{ status: string }>(`/vaults/${op.vaultId}/items/${targetId}`, {
              method: 'DELETE'
            });
            syncedCount++;
          } else if (op.op === 'move') {
            const targetId = idMap.get(op.itemId) ?? op.itemId;
            await this.request<{ status: string }>(`/vaults/${op.sourceVaultId}/items/${targetId}/move`, {
              method: 'PUT',
              body: JSON.stringify({ target_vault_id: op.targetVaultId })
            });
            syncedCount++;
          }
        } catch (err) {
          if (this.isNetworkError(err)) {
            remaining.push(...queue.slice(i));
            break;
          }
          // Non-network error (e.g. item already deleted): drop stale op
        }
      }
      this.writeSyncQueue(remaining);
      if (syncedCount > 0) {
        this.onSyncListeners.forEach((cb) => cb(syncedCount));
      }
      return syncedCount;
    } finally {
      this.isFlushing = false;
    }
  }

  async syncStandaloneToGateway(): Promise<number> {
    if (this.standalone || !this.token || typeof window === 'undefined') return 0;

    const localVaults = this.readJson<ApiVault[]>('opentermius_local_vaults', []);
    const vaultIds = localVaults.length > 0 ? localVaults.map((v) => v.id) : ['local-vault-primary'];

    const localItemsByVault = vaultIds
      .map((id) => ({
        vaultId: id,
        items: this.readJson<ApiVaultItem[]>(`opentermius_local_items_${id}`, [])
      }))
      .filter((entry) => entry.items.length > 0);

    if (localItemsByVault.length === 0) return 0;

    let gwVaults = await this.request<ApiVault[]>('/vaults');
    let personalVault = gwVaults.find((v) => v.type === 'personal') ?? gwVaults[0];
    if (!personalVault) {
      personalVault = await this.request<ApiVault>('/vaults', {
        method: 'POST',
        body: JSON.stringify({
          name: 'Personal Cloud Vault',
          type: 'personal',
          encrypted_key: 'initial-master-vault-key-aes-256'
        })
      });
    }

    const existingItems = await this.request<ApiVaultItem[]>(`/vaults/${personalVault.id}/items`);
    const existingKeys = new Set(existingItems.map((i) => `${i.kind}:${i.label}:${i.ciphertext}`));

    let migratedCount = 0;
    for (const entry of localItemsByVault) {
      const remainingLocal: ApiVaultItem[] = [];
      for (const item of entry.items) {
        const dedupeKey = `${item.kind}:${item.label}:${item.ciphertext}`;
        if (!existingKeys.has(dedupeKey)) {
          try {
            await this.request<ApiVaultItem>(`/vaults/${personalVault.id}/items`, {
              method: 'POST',
              body: JSON.stringify({
                kind: item.kind,
                label: item.label,
                ciphertext: item.ciphertext,
                nonce: item.nonce
              })
            });
            existingKeys.add(dedupeKey);
            migratedCount++;
          } catch (err) {
            if (this.isNetworkError(err)) {
              remainingLocal.push(item);
            }
          }
        }
      }
      if (remainingLocal.length === 0) {
        localStorage.removeItem(`opentermius_local_items_${entry.vaultId}`);
      } else {
        this.writeJson(`opentermius_local_items_${entry.vaultId}`, remainingLocal);
      }
    }

    return migratedCount;
  }

  setGateway(apiBase: string) {
    this.apiBase = apiBase.replace(/\/+$/, '');
    if (typeof window !== 'undefined') {
      localStorage.setItem('opentermius_api_base', this.apiBase);
    }
  }

  getApiBase(): string {
    return this.apiBase;
  }

  isStandalone(): boolean {
    return this.standalone;
  }

  setStandalone(enabled: boolean) {
    this.standalone = enabled;
    if (typeof window !== 'undefined') {
      if (enabled) {
        localStorage.setItem('opentermius_mode', 'standalone');
      } else {
        localStorage.removeItem('opentermius_mode');
      }
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('opentermius_jwt_token', token);
    }
  }

  getToken(): string | null {
    return this.token;
  }

  clearToken() {
    this.token = null;
    this.standalone = false;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('opentermius_jwt_token');
      localStorage.removeItem('opentermius_mode');
    }
  }

  private async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers || {});
    headers.set('Content-Type', 'application/json');

    if (this.token) {
      headers.set('Authorization', `Bearer ${this.token}`);
    }

    const response = await fetch(`${this.apiBase}${path}`, {
      ...options,
      headers
    });

    if (!response.ok) {
      let errorMsg = `API Error: ${response.status} ${response.statusText}`;
      try {
        const errJson = await response.json();
        if (errJson.error) errorMsg = errJson.error;
      } catch {
        // ignore
      }
      throw new Error(errorMsg);
    }

    return response.json() as Promise<T>;
  }

  // Auth Endpoints (Registration is handled exclusively by Gateway Web Console)
  async login(email: string, password: string): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    });
    this.setToken(res.token);
    this.setStandalone(false);
    return res;
  }

  async getMe() {
    if (this.standalone) {
      return { id: 'standalone-local-user', email: 'local@device', name: 'Standalone Operator' };
    }
    try {
      const me = await this.request<{ id: string; email: string; name: string }>('/auth/me');
      this.writeJson(GW_USER_CACHE_KEY, me);
      return me;
    } catch (err) {
      if (this.isNetworkError(err)) {
        const cached = this.readJson<{ id: string; email: string; name: string } | null>(GW_USER_CACHE_KEY, null);
        if (cached && this.token) return cached;
      }
      throw err;
    }
  }

  // Vault Endpoints
  async getVaults(): Promise<ApiVault[]> {
    if (this.standalone) {
      const raw = localStorage.getItem('opentermius_local_vaults');
      if (raw) {
        try {
          return JSON.parse(raw);
        } catch {}
      }
      const initialVault: ApiVault = {
        id: 'local-vault-primary',
        owner_id: 'standalone-local-user',
        name: 'Local Standalone Vault',
        type: 'personal',
        encrypted_key: 'local-master-vault-key-aes-256',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      localStorage.setItem('opentermius_local_vaults', JSON.stringify([initialVault]));
      return [initialVault];
    }
    try {
      await this.flushSyncQueue();
      const vaults = await this.request<ApiVault[]>('/vaults');
      this.writeJson(GW_VAULTS_CACHE_KEY, vaults);
      return vaults;
    } catch (err) {
      if (this.isNetworkError(err)) {
        const cached = this.readJson<ApiVault[]>(GW_VAULTS_CACHE_KEY, []);
        if (cached.length > 0) return cached;
      }
      throw err;
    }
  }

  async createVault(name: string, type: 'personal' | 'team', encryptedKey: string): Promise<ApiVault> {
    if (this.standalone) {
      const vaults = await this.getVaults();
      const newVault: ApiVault = {
        id: crypto.randomUUID(),
        owner_id: 'standalone-local-user',
        name,
        type,
        encrypted_key: encryptedKey,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      vaults.push(newVault);
      localStorage.setItem('opentermius_local_vaults', JSON.stringify(vaults));
      return newVault;
    }
    return this.request<ApiVault>('/vaults', {
      method: 'POST',
      body: JSON.stringify({ name, type, encrypted_key: encryptedKey })
    });
  }

  async getVaultItems(vaultId: string, kind?: string): Promise<ApiVaultItem[]> {
    if (this.standalone) {
      const raw = localStorage.getItem(`opentermius_local_items_${vaultId}`);
      let items: ApiVaultItem[] = [];
      if (raw) {
        try {
          items = JSON.parse(raw);
        } catch {}
      }
      if (kind) {
        return items.filter((i) => i.kind === kind);
      }
      return items;
    }
    try {
      await this.flushSyncQueue();
      const allItems = await this.request<ApiVaultItem[]>(`/vaults/${vaultId}/items`);
      this.writeGatewayItemsCache(vaultId, allItems);
      return kind ? allItems.filter((i) => i.kind === kind) : allItems;
    } catch (err) {
      if (this.isNetworkError(err)) {
        const cached = this.readGatewayItemsCache(vaultId);
        return kind ? cached.filter((i) => i.kind === kind) : cached;
      }
      throw err;
    }
  }

  async createVaultItem(
    vaultId: string,
    item: {
      kind: 'host' | 'key' | 'identity' | 'snippet' | 'tunnel';
      label: string;
      ciphertext: string;
      nonce: string;
    }
  ): Promise<ApiVaultItem> {
    if (this.standalone) {
      const items = await this.getVaultItems(vaultId);
      const created: ApiVaultItem = {
        id: crypto.randomUUID(),
        vault_id: vaultId,
        kind: item.kind,
        label: item.label,
        ciphertext: item.ciphertext,
        nonce: item.nonce,
        version: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      items.unshift(created);
      localStorage.setItem(`opentermius_local_items_${vaultId}`, JSON.stringify(items));
      return created;
    }
    try {
      const created = await this.request<ApiVaultItem>(`/vaults/${vaultId}/items`, {
        method: 'POST',
        body: JSON.stringify(item)
      });
      const cached = this.readGatewayItemsCache(vaultId);
      this.writeGatewayItemsCache(vaultId, [created, ...cached]);
      return created;
    } catch (err) {
      if (!this.isNetworkError(err)) throw err;
      const tempId = `local-pending-${crypto.randomUUID()}`;
      const optimistic: ApiVaultItem = {
        id: tempId,
        vault_id: vaultId,
        kind: item.kind,
        label: item.label,
        ciphertext: item.ciphertext,
        nonce: item.nonce,
        version: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      const cached = this.readGatewayItemsCache(vaultId);
      this.writeGatewayItemsCache(vaultId, [optimistic, ...cached]);
      this.enqueueOfflineOp({ op: 'create', vaultId, tempId, item });
      return optimistic;
    }
  }

  async updateVaultItem(
    vaultId: string,
    itemId: string,
    item: {
      label: string;
      ciphertext: string;
      nonce: string;
    }
  ): Promise<ApiVaultItem> {
    if (this.standalone) {
      const items = await this.getVaultItems(vaultId);
      const idx = items.findIndex((i) => i.id === itemId);
      if (idx !== -1) {
        items[idx] = {
          ...items[idx],
          label: item.label,
          ciphertext: item.ciphertext,
          nonce: item.nonce,
          version: items[idx].version + 1,
          updated_at: new Date().toISOString()
        };
        localStorage.setItem(`opentermius_local_items_${vaultId}`, JSON.stringify(items));
        return items[idx];
      }
    }
    try {
      const updated = await this.request<ApiVaultItem>(`/vaults/${vaultId}/items/${itemId}`, {
        method: 'PUT',
        body: JSON.stringify(item)
      });
      const cached = this.readGatewayItemsCache(vaultId);
      this.writeGatewayItemsCache(
        vaultId,
        cached.map((c) => (c.id === itemId ? updated : c))
      );
      return updated;
    } catch (err) {
      if (!this.isNetworkError(err)) throw err;
      const cached = this.readGatewayItemsCache(vaultId);
      const idx = cached.findIndex((c) => c.id === itemId);
      const optimistic: ApiVaultItem = idx !== -1
        ? {
            ...cached[idx],
            label: item.label,
            ciphertext: item.ciphertext,
            nonce: item.nonce,
            version: cached[idx].version + 1,
            updated_at: new Date().toISOString()
          }
        : {
            id: itemId,
            vault_id: vaultId,
            kind: 'host',
            label: item.label,
            ciphertext: item.ciphertext,
            nonce: item.nonce,
            version: 1,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
          };
      if (idx !== -1) {
        cached[idx] = optimistic;
      } else {
        cached.unshift(optimistic);
      }
      this.writeGatewayItemsCache(vaultId, cached);
      this.enqueueOfflineOp({ op: 'update', vaultId, itemId, item });
      return optimistic;
    }
  }

  async deleteVaultItem(vaultId: string, itemId: string): Promise<{ status: string }> {
    if (this.standalone) {
      const items = await this.getVaultItems(vaultId);
      const filtered = items.filter((i) => i.id !== itemId);
      localStorage.setItem(`opentermius_local_items_${vaultId}`, JSON.stringify(filtered));
      return { status: 'deleted' };
    }
    try {
      const res = await this.request<{ status: string }>(`/vaults/${vaultId}/items/${itemId}`, {
        method: 'DELETE'
      });
      const cached = this.readGatewayItemsCache(vaultId);
      this.writeGatewayItemsCache(vaultId, cached.filter((c) => c.id !== itemId));
      return res;
    } catch (err) {
      if (!this.isNetworkError(err)) throw err;
      const cached = this.readGatewayItemsCache(vaultId);
      this.writeGatewayItemsCache(vaultId, cached.filter((c) => c.id !== itemId));
      this.enqueueOfflineOp({ op: 'delete', vaultId, itemId });
      return { status: 'queued' };
    }
  }

  async moveVaultItem(sourceVaultId: string, itemId: string, targetVaultId: string): Promise<{ status: string }> {
    if (this.standalone) {
      const sourceItems = await this.getVaultItems(sourceVaultId);
      const targetItems = await this.getVaultItems(targetVaultId);
      const itemToMove = sourceItems.find((i) => i.id === itemId);
      if (itemToMove) {
        const updatedSource = sourceItems.filter((i) => i.id !== itemId);
        localStorage.setItem(`opentermius_local_items_${sourceVaultId}`, JSON.stringify(updatedSource));
        itemToMove.vault_id = targetVaultId;
        targetItems.unshift(itemToMove);
        localStorage.setItem(`opentermius_local_items_${targetVaultId}`, JSON.stringify(targetItems));
      }
      return { status: 'moved' };
    }
    if (itemId.startsWith('local-pending-')) {
      const sourceCached = this.readGatewayItemsCache(sourceVaultId);
      const itemToMove = sourceCached.find((i) => i.id === itemId);
      this.writeGatewayItemsCache(sourceVaultId, sourceCached.filter((i) => i.id !== itemId));
      if (itemToMove) {
        const targetCached = this.readGatewayItemsCache(targetVaultId);
        this.writeGatewayItemsCache(targetVaultId, [{ ...itemToMove, vault_id: targetVaultId }, ...targetCached]);
      }
      this.enqueueOfflineOp({ op: 'move', sourceVaultId, itemId, targetVaultId });
      void this.flushSyncQueue();
      return { status: 'moved' };
    }
    try {
      await this.flushSyncQueue();
      const res = await this.request<{ status: string }>(`/vaults/${sourceVaultId}/items/${itemId}/move`, {
        method: 'PUT',
        body: JSON.stringify({ target_vault_id: targetVaultId })
      });
      const sourceCached = this.readGatewayItemsCache(sourceVaultId);
      const itemToMove = sourceCached.find((i) => i.id === itemId);
      this.writeGatewayItemsCache(sourceVaultId, sourceCached.filter((i) => i.id !== itemId));
      if (itemToMove) {
        const targetCached = this.readGatewayItemsCache(targetVaultId);
        this.writeGatewayItemsCache(targetVaultId, [{ ...itemToMove, vault_id: targetVaultId }, ...targetCached]);
      }
      return res;
    } catch (err) {
      if (!this.isNetworkError(err)) throw err;
      const sourceCached = this.readGatewayItemsCache(sourceVaultId);
      const itemToMove = sourceCached.find((i) => i.id === itemId);
      this.writeGatewayItemsCache(sourceVaultId, sourceCached.filter((i) => i.id !== itemId));
      if (itemToMove) {
        const targetCached = this.readGatewayItemsCache(targetVaultId);
        this.writeGatewayItemsCache(targetVaultId, [{ ...itemToMove, vault_id: targetVaultId }, ...targetCached]);
      }
      this.enqueueOfflineOp({ op: 'move', sourceVaultId, itemId, targetVaultId });
      return { status: 'queued' };
    }
  }

  // Vault Members & Team Collaboration
  async getVaultMembers(vaultId: string): Promise<ApiVaultMember[]> {
    if (this.standalone) {
      return [{ id: 'mem-1', vault_id: vaultId, user_id: 'standalone-local-user', role: 'owner', created_at: new Date().toISOString() }];
    }
    return this.request<ApiVaultMember[]>(`/vaults/${vaultId}/members`);
  }

  async inviteVaultMember(vaultId: string, email: string, role: string = 'member'): Promise<ApiVaultMember> {
    if (this.standalone) {
      throw new Error('Team invites are only available when connected to a Gateway sync server.');
    }
    return this.request<ApiVaultMember>(`/vaults/${vaultId}/members`, {
      method: 'POST',
      body: JSON.stringify({ email, role })
    });
  }

  // Native SFTP File Manager Operations (via Tauri Rust russh-sftp)
  private toNativeSftpConfig(config: SftpHostConfig) {
    return {
      address: config.address,
      port: config.port || 22,
      username: config.username,
      password: config.password || '',
      privateKey: config.privateKey || '',
      passphrase: config.passphrase || ''
    };
  }

  async sftpList(config: SftpHostConfig, path: string = ''): Promise<{ path: string; items: ApiSftpFile[] }> {
    const { invoke } = await import('@tauri-apps/api/core');
    return invoke<{ path: string; items: ApiSftpFile[] }>('sftp_list', {
      config: this.toNativeSftpConfig(config),
      path
    });
  }

  async sftpMkdir(config: SftpHostConfig, path: string): Promise<void> {
    const { invoke } = await import('@tauri-apps/api/core');
    await invoke('sftp_mkdir', {
      config: this.toNativeSftpConfig(config),
      path
    });
  }

  async sftpDelete(config: SftpHostConfig, path: string): Promise<void> {
    const { invoke } = await import('@tauri-apps/api/core');
    await invoke('sftp_delete', {
      config: this.toNativeSftpConfig(config),
      path
    });
  }

  async sftpDownloadBlob(config: SftpHostConfig, path: string): Promise<Blob> {
    const { invoke } = await import('@tauri-apps/api/core');
    const bytes = await invoke<number[]>('sftp_read_bytes', {
      config: this.toNativeSftpConfig(config),
      path
    });
    return new Blob([new Uint8Array(bytes)]);
  }

  async sftpDownload(config: SftpHostConfig, path: string, filename: string): Promise<void> {
    const blob = await this.sftpDownloadBlob(config, path);
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename || 'downloaded-file';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  }

  async sftpUpload(config: SftpHostConfig, remotePath: string, file: File): Promise<void> {
    const { invoke } = await import('@tauri-apps/api/core');
    const base = remotePath.endsWith('/') ? remotePath.slice(0, -1) : remotePath;
    const targetPath = `${base}/${file.name}`;
    const bytes = Array.from(new Uint8Array(await file.arrayBuffer()));
    await invoke('sftp_write_bytes', {
      config: this.toNativeSftpConfig(config),
      path: targetPath,
      data: bytes
    });
  }
}

export const api = new ApiClient();
