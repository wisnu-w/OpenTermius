// Reactive Platform & Device Form-Factor Detection (Svelte 5 Runes)

export type LayoutMode = 'mobile' | 'tablet' | 'desktop';

class PlatformStore {
  width = $state<number>(typeof window !== 'undefined' ? window.innerWidth : 1280);
  height = $state<number>(typeof window !== 'undefined' ? window.innerHeight : 800);
  visualHeight = $state<number>(
    typeof window !== 'undefined' ? (window.visualViewport?.height ?? window.innerHeight) : 800
  );
  userAgent = $state<string>(typeof navigator !== 'undefined' ? navigator.userAgent : '');
  isTouch = $state<boolean>(
    typeof window !== 'undefined' &&
      ('ontouchstart' in window || navigator.maxTouchPoints > 0)
  );

  isAndroid = $derived(/android/i.test(this.userAgent));
  isIOS = $derived(/ipad|iphone|ipod/i.test(this.userAgent));
  isMac = $derived(/macintosh|mac os x/i.test(this.userAgent) && !this.isIOS);
  isNativeTauri = $derived(
    typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
  );

  layoutMode = $derived.by<LayoutMode>(() => {
    if (this.width < 768) return 'mobile';
    if (this.width < 1024 || (this.isTouch && this.width <= 1180 && (this.isAndroid || this.isIOS))) {
      return 'tablet';
    }
    return 'desktop';
  });

  isMobile = $derived(this.layoutMode === 'mobile');
  isTablet = $derived(this.layoutMode === 'tablet');
  isDesktop = $derived(this.layoutMode === 'desktop');
  isHandheld = $derived(this.layoutMode === 'mobile' || this.layoutMode === 'tablet');
  isKeyboardOpen = $derived(
    this.isHandheld && this.height > 0 && this.visualHeight < this.height * 0.78
  );

  constructor() {
    if (typeof window !== 'undefined') {
      const update = () => {
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.visualHeight = window.visualViewport?.height ?? window.innerHeight;
        this.userAgent = navigator.userAgent;
        this.isTouch = 'ontouchstart' in window || navigator.maxTouchPoints > 0;
      };
      const updateViewport = () => {
        this.visualHeight = window.visualViewport?.height ?? window.innerHeight;
      };
      window.addEventListener('resize', update);
      window.addEventListener('orientationchange', () => setTimeout(update, 100));
      window.visualViewport?.addEventListener('resize', updateViewport);
    }
  }
}

export const platform = new PlatformStore();
