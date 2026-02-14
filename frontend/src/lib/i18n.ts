import ja from '../i18n/ja.json';
import en from '../i18n/en.json';

export type Locale = 'ja' | 'en';

const messages: Record<Locale, Record<string, unknown>> = {
  ja: ja as Record<string, unknown>,
  en: en as Record<string, unknown>,
};

function getNested(obj: Record<string, unknown>, path: string): string | undefined {
  const keys = path.split('.');
  let current: unknown = obj;
  for (const key of keys) {
    if (current && typeof current === 'object' && key in current) {
      current = (current as Record<string, unknown>)[key];
    } else {
      return undefined;
    }
  }
  return typeof current === 'string' ? current : undefined;
}

function replaceParams(text: string, params: Record<string, string | number>): string {
  return text.replace(/\{(\w+)\}/g, (_, key) => String(params[key] ?? `{${key}}`));
}

export function t(locale: Locale, key: string, params?: Record<string, string | number>): string {
  const text = getNested(messages[locale] as Record<string, unknown>, key) ?? getNested(messages.ja as Record<string, unknown>, key) ?? key;
  return params ? replaceParams(text, params) : text;
}

export const locales: Locale[] = ['ja', 'en'];
export const defaultLocale: Locale = 'ja';
