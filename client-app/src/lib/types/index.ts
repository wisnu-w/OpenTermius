// OpenTermius TypeScript Contracts (Matches Go Domain)

export type VaultType = 'personal' | 'team';
export type VaultRole = 'owner' | 'admin' | 'member';
export type ItemKind = 'host' | 'key' | 'identity' | 'snippet' | 'tunnel';

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export interface Vault {
  id: string;
  owner_id: string;
  name: string;
  type: VaultType;
  encrypted_key: string;
  created_at: string;
  updated_at: string;
}

export interface VaultMember {
  id: string;
  vault_id: string;
  user_id: string;
  role: VaultRole;
  encrypted_vault_key: string;
  created_at: string;
}

export interface VaultItem {
  id: string;
  vault_id: string;
  kind: ItemKind;
  label: string;
  ciphertext: string;
  nonce: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface HostConfig {
  address: string;
  port: number;
  username: string;
  identity_id?: string;
  key_id?: string;
  startup_snippet_id?: string;
  use_mosh?: boolean;
}

export interface SnippetConfig {
  script: string;
  auto_run: boolean;
}

export interface SessionLog {
  id: string;
  vault_id: string;
  user_id: string;
  host_label: string;
  started_at: string;
  ended_at?: string;
  recorded_data: string;
}
