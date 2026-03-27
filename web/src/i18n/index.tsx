import React, { createContext, useContext, useMemo, useState } from 'react';
import zhCN from './locales/zh-CN';
import enUS from './locales/en-US';

type Lang = 'zh-CN' | 'en-US';

type I18nContextValue = {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: (key: string) => string;
};

const dictionaries: Record<Lang, Record<string, string>> = {
  'zh-CN': zhCN as Record<string, string>,
  'en-US': enUS as Record<string, string>,
};

const I18nContext = createContext<I18nContextValue>({
  lang: 'zh-CN',
  setLang: () => undefined,
  t: (key) => key,
});

export const I18nProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [lang, setLangState] = useState<Lang>(() => {
    const saved = localStorage.getItem('lang');
    if (saved === 'en-US' || saved === 'zh-CN') return saved;
    return 'zh-CN';
  });
  const setLang = (next: Lang) => {
    localStorage.setItem('lang', next);
    setLangState(next);
  };
  const value = useMemo(() => ({
    lang,
    setLang,
    t: (key: string) => dictionaries[lang][key] || dictionaries['zh-CN'][key] || key,
  }), [lang]);
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
};

export const useI18n = () => useContext(I18nContext);
