/**
 * The rail's version line, one format fleet-wide: "v<semver>".
 *
 * A development build reports `git describe` output ("v0.285.3-31-gf4194afb-dirty"):
 * the release it grew from, the commits since, the commit and a dirty flag.
 * Only the release belongs in the visible line, which is 224px wide and read at
 * a glance; the rest is for whoever is debugging, so it goes in the tooltip.
 * A prerelease tag (`-rc.1`, `-beta.2`) is part of the version and stays.
 */

const VERSION = /^v?(\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?)(?![\w.])/;

/** The visible text for a raw version string; a non-semver value ("dev") is shown as it is. */
export function formatVersion(raw: string): string {
  const match = VERSION.exec(raw.trim());
  return match?.[1] === undefined ? raw.trim() : `v${match[1]}`;
}

/** The raw string, when it carries more than the visible line does. */
export function buildDetail(raw: string): string | undefined {
  const trimmed = raw.trim();
  return trimmed === formatVersion(trimmed) ? undefined : trimmed;
}
