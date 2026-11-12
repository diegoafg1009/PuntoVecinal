declare module '#app' {
  interface PageMeta {
    /** Hides the mobile top and bottom bars; the desktop header stays. */
    focusedOnMobile?: boolean
    /** Replaces the mobile app top bar with one the page renders; the bottom bar stays. */
    ownTopBarOnMobile?: boolean
  }
}

export {}
