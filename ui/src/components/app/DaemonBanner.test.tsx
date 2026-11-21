import { render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../../i18n';
import { DaemonBanner } from './DaemonBanner';

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('DaemonBanner', () => {
  it('says the daemon is unreachable and is retrying', async () => {
    await i18n.changeLanguage('en');
    render(<DaemonBanner />);
    expect(screen.getByRole('alert')).toHaveTextContent("Can't reach Seed daemon, retrying");
  });

  it('is translated in Spanish', async () => {
    await i18n.changeLanguage('es');
    render(<DaemonBanner />);
    expect(screen.getByRole('alert')).toHaveTextContent(
      'No se puede conectar con el daemon de Seed, reintentando',
    );
  });
});
