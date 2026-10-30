import { createInstance } from "i18next";
import { initReactI18next } from "react-i18next";
import common from "../i18n/locales/en/common.json";
import components from "../i18n/locales/en/components.json";
import pages from "../i18n/locales/en/pages.json";

const i18n = createInstance();
void i18n.use(initReactI18next).init({
  lng: "en",
  fallbackLng: "en",
  defaultNS: "common",
  resources: { en: { common, components, pages } },
  interpolation: { escapeValue: false },
});

export default i18n;
