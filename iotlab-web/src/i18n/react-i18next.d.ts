import type { en } from "./locales/en";

// Typed t(): keys are checked against the English catalog.
declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "translation";
    resources: {
      translation: typeof en;
    };
  }
}
