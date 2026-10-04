/**
 * Button.i18n.test.tsx — the loading spinner's accessible title is locale copy.
 *
 * The spinner's `<title>` was the bare word "Loading", single-word JSX text the
 * shared gate skipped until .github#100. A screen reader reads it out on every
 * busy button, so a Spanish operator heard English.
 */
import { render } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { Button } from './Button';

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('Button — real locale copy', () => {
  it('titles the loading spinner in Spanish under es', async () => {
    await i18n.changeLanguage('es');
    const { container } = render(<Button loading>Guardar</Button>);

    expect(container.querySelector('svg title')).toHaveTextContent('Cargando...');
  });
});
