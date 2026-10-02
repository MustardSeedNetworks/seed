/**
 * Spanish resources, split from the entry bundle: i18n/index.ts imports this
 * module dynamically, so only a session running in Spanish downloads it.
 */
import esCards from '@locales/es/cards.json';
import esCommon from '@locales/es/common.json';
import esErrors from '@locales/es/errors.json';
import esGlossary from '@locales/es/glossary.json';
import esHelp from '@locales/es/help.json';
import esPages from '@locales/es/pages.json';
import esSettings from '@locales/es/settings.json';
import esSetup from '@locales/es/setup.json';
import type { ResourceLanguage } from 'i18next';

export const es: ResourceLanguage = {
  common: esCommon,
  cards: esCards,
  settings: esSettings,
  errors: esErrors,
  glossary: esGlossary,
  help: esHelp,
  pages: esPages,
  setup: esSetup,
};
