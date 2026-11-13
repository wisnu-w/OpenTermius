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

class ApiClient {
  private token: string | null = null;
  private apiBase: string = 'http://localhost:8080/api/v1';
  private wsBase: string = 'ws://localhost:8080/ws';
  private standalone: boolean = false;

  constructor() {
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('opentermius_jwt_token');
      this.apiBase = localStorage.getItem('opentermius_api_base') || 'http://localhost:8080/api/v1';
      this.wsBase = localStorage.getItem('opentermius_ws_base') || 'ws://localhost:8080/ws';
      this.standalone = localStorage.getItem('opentermius_mode') === 'standalone';
    }
  }

  setGateway(apiBase: string, wsBase: string) {
    this.apiBase = apiBase.replace(/\/+$/, '');
    this.wsBase = wsBase.replace(/\/+$/, '');
    if (typeof window !== 'undefined') {
      localStorage.setItem('opentermius_api_base', this.apiBase);
      localStorage.setItem('opentermius_ws_base', this.wsBase);
    }
  }

  getApiBase(): string {
    return this.apiBase;
  }

  getWsBase(): string {
    return this.wsBase;
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

  // Auth Endpoints
  async register(email: string, password: string, name: string): Promise<AuthResponse> {
    const res = await this.request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name })
    });
    this.setToken(res.token);
    this.setStandalone(false);
    return res;
  }

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
    return this.request<{ id: string; email: string; name: string }>('/auth/me');
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
    return this.request<ApiVault[]>('/vaults');
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
    const query = kind ? `?kind=${encodeURIComponent(kind)}` : '';
    return this.request<ApiVaultItem[]>(`/vaults/${vaultId}/items${query}`);
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
    return this.request<ApiVaultItem>(`/vaults/${vaultId}/items`, {
      method: 'POST',
      body: JSON.stringify(item)
    });
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
    return this.request<ApiVaultItem>(`/vaults/${vaultId}/items/${itemId}`, {
      method: 'PUT',
      body: JSON.stringify(item)
    });
  }

  async deleteVaultItem(vaultId: string, itemId: string): Promise<{ status: string }> {
    if (this.standalone) {
      const items = await this.getVaultItems(vaultId);
      const filtered = items.filter((i) => i.id !== itemId);
      localStorage.setItem(`opentermius_local_items_${vaultId}`, JSON.stringify(filtered));
      return { status: 'deleted' };
    }
    return this.request<{ status: string }>(`/vaults/${vaultId}/items/${itemId}`, {
      method: 'DELETE'
    });
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
    return this.request<{ status: string }>(`/vaults/${sourceVaultId}/items/${itemId}/move`, {
      method: 'PUT',
      body: JSON.stringify({ target_vault_id: targetVaultId })
    });
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

  // Terminal WebSocket URL
  getTerminalWsUrl(hostId: string): string {
    const token = this.token || '';
    return `${this.wsBase}/terminal/${hostId}?token=${encodeURIComponent(token)}`;
  }

  // SFTP File Manager Operations
  async sftpList(config: SftpHostConfig, path: string = ''): Promise<{ path: string; items: ApiSftpFile[] }> {
    const payload = {
      host: config.address,
      port: config.port || 22,
      username: config.username,
      authType: config.authType || 'password',
      password: config.password || '',
      privateKey: config.privateKey || '',
      passphrase: config.passphrase || '',
      path
    };

    const token = this.token || 'standalone';
    const response = await fetch(`${this.apiBase}/sftp/list`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'SFTP connection failed' }));
      throw new Error(err.error || 'SFTP connection failed');
    }
    return response.json();
  }

  async sftpMkdir(config: SftpHostConfig, path: string): Promise<void> {
    const payload = {
      host: config.address,
      port: config.port || 22,
      username: config.username,
      authType: config.authType || 'password',
      password: config.password || '',
      privateKey: config.privateKey || '',
      passphrase: config.passphrase || '',
      path
    };

    const token = this.token || 'standalone';
    const response = await fetch(`${this.apiBase}/sftp/mkdir`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'Failed to create directory' }));
      throw new Error(err.error || 'Failed to create directory');
    }
  }

  async sftpDelete(config: SftpHostConfig, path: string): Promise<void> {
    const payload = {
      host: config.address,
      port: config.port || 22,
      username: config.username,
      authType: config.authType || 'password',
      password: config.password || '',
      privateKey: config.privateKey || '',
      passphrase: config.passphrase || '',
      path
    };

    const token = this.token || 'standalone';
    const response = await fetch(`${this.apiBase}/sftp/delete`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'Failed to delete remote item' }));
      throw new Error(err.error || 'Failed to delete remote item');
    }
  }

  async sftpDownload(config: SftpHostConfig, path: string, filename: string): Promise<void> {
    const payload = {
      host: config.address,
      port: config.port || 22,
      username: config.username,
      authType: config.authType || 'password',
      password: config.password || '',
      privateKey: config.privateKey || '',
      passphrase: config.passphrase || '',
      path
    };

    const token = this.token || 'standalone';
    const response = await fetch(`${this.apiBase}/sftp/download`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'Download failed' }));
      throw new Error(err.error || 'Download failed');
    }

    const blob = await response.blob();
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
    const formData = new FormData();
    formData.append('host', config.address);
    formData.append('port', (config.port || 22).toString());
    formData.append('username', config.username);
    formData.append('authType', config.authType || 'password');
    formData.append('password', config.password || '');
    formData.append('privateKey', config.privateKey || '');
    formData.append('passphrase', config.passphrase || '');
    formData.append('remotePath', remotePath);
    formData.append('file', file);

    const token = this.token || 'standalone';
    const response = await fetch(`${this.apiBase}/sftp/upload`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      },
      body: formData
    });

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'Upload failed' }));
      throw new Error(err.error || 'Upload failed');
    }
  }
}

export const api = new ApiClient();
