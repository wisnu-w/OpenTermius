// Svelte 5 Terminal State Management with Runes

export interface TerminalHostConfig {
  address: string;
  port: number;
  username: string;
  authType?: 'password' | 'key';
  password?: string;
  privateKey?: string;
  passphrase?: string;
}

export interface TerminalTab {
  id: string;
  hostId: string;
  label: string;
  config?: TerminalHostConfig;
  connected: boolean;
  active: boolean;
}

class TerminalStore {
  // Svelte 5 Runes
  tabs = $state<TerminalTab[]>([]);
  activeTabId = $state<string | null>(null);
  splitTabId = $state<string | null>(null);
  splitDirection = $state<'horizontal' | 'vertical'>('horizontal');

  activeTab = $derived(
    this.tabs.find((tab) => tab.id === this.activeTabId) || null
  );

  splitTab = $derived(
    this.tabs.find((tab) => tab.id === this.splitTabId) || null
  );

  tabCount = $derived(this.tabs.length);

  openTab(hostId: string, label: string, config?: TerminalHostConfig): string {
    const id = crypto.randomUUID();
    const newTab: TerminalTab = {
      id,
      hostId,
      label,
      config,
      connected: false,
      active: true
    };

    // Deactivate previous active tab
    for (const tab of this.tabs) {
      tab.active = false;
    }

    this.tabs.push(newTab);
    this.activeTabId = id;
    return id;
  }

  splitCurrent(hostId: string, label: string, config?: TerminalHostConfig, direction: 'horizontal' | 'vertical' = 'horizontal'): string {
    const id = crypto.randomUUID();
    const newTab: TerminalTab = {
      id,
      hostId,
      label,
      config,
      connected: false,
      active: true
    };
    this.tabs.push(newTab);
    this.splitTabId = id;
    this.splitDirection = direction;
    return id;
  }

  setSplit(tabId: string | null, direction: 'horizontal' | 'vertical' = 'horizontal') {
    this.splitTabId = tabId;
    this.splitDirection = direction;
  }

  closeSplit() {
    if (this.splitTabId) {
      const idToClose = this.splitTabId;
      this.splitTabId = null;
      this.closeTab(idToClose);
    }
  }

  closeTab(tabId: string) {
    if (this.splitTabId === tabId) {
      this.splitTabId = null;
    }
    const idx = this.tabs.findIndex((t) => t.id === tabId);
    if (idx !== -1) {
      this.tabs.splice(idx, 1);
      if (this.activeTabId === tabId) {
        this.activeTabId = this.tabs.length > 0 ? this.tabs[Math.max(0, idx - 1)].id : null;
        if (this.activeTabId) {
          const nextActive = this.tabs.find((t) => t.id === this.activeTabId);
          if (nextActive) nextActive.active = true;
        }
      }
    }
  }

  setConnected(tabId: string, connected: boolean) {
    const tab = this.tabs.find((t) => t.id === tabId);
    if (tab) {
      tab.connected = connected;
    }
  }

  selectTab(tabId: string) {
    for (const tab of this.tabs) {
      tab.active = tab.id === tabId;
    }
    this.activeTabId = tabId;
  }
}

export const terminalStore = new TerminalStore();
