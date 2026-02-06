import { createI18n } from "vue-i18n";
import en from "../locales/en.json";
import zhHans from "../locales/zh-Hans.json";

const i18n = createI18n({
  legacy: false, // use Composition API
  locale: navigator.language.startsWith("zh") ? "zh-Hans" : "en",
  fallbackLocale: "en",
  messages: {
    en,
    "zh-Hans": zhHans,
  },
});

export default i18n;
