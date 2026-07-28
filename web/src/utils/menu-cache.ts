const MENU_CACHE_VERSION = 'v6'
const MENU_CACHE_TTL = 12 * 60 * 60 * 1000

export const getMenuCacheUsername = (userInfo?: any) => {
  return userInfo?.username || 'anonymous'
}

const getMenuCacheKey = (username: string) => {
  return `opshub:user-menus:${MENU_CACHE_VERSION}:${username || 'anonymous'}`
}

export const readUserMenuCache = (username: string) => {
  try {
    const raw = localStorage.getItem(getMenuCacheKey(username))
    if (!raw) return null

    const cache = JSON.parse(raw)
    if (!cache || Date.now() - Number(cache.timestamp || 0) > MENU_CACHE_TTL) {
      localStorage.removeItem(getMenuCacheKey(username))
      return null
    }

    return Array.isArray(cache.menus) ? cache.menus : null
  } catch {
    localStorage.removeItem(getMenuCacheKey(username))
    return null
  }
}

export const writeUserMenuCache = (username: string, menus: any[]) => {
  try {
    localStorage.setItem(getMenuCacheKey(username), JSON.stringify({
      timestamp: Date.now(),
      menus,
    }))
  } catch {
    // 菜单缓存失败不影响正常使用
  }
}
