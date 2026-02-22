/** Base URL of this site instance (no trailing slash). Used for OGP absolute URLs. */
export const SITE_URL: string =
  import.meta.env.PUBLIC_SITE_URL ||
  import.meta.env.PUBLIC_OFFICIAL_URL ||
  'http://localhost:4321';
