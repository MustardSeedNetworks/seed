/**
 * HelpSectionBody.tsx
 *
 * Generic renderer for a HelpSection's typed `HelpBlock[]`. One switch on the
 * block `kind` covers every section, so adding or editing content only means
 * editing `helpDrawerContent.tsx` — no new JSX. Uses Seed theme tokens only.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 */

import type { ReactElement } from 'react';
import { cn, layout, radius, spacing } from '../../styles/theme';
import type { HelpBlock } from './helpDrawerContent';

interface HelpSectionBodyProps {
  blocks: HelpBlock[];
}

export function HelpSectionBody({ blocks }: HelpSectionBodyProps): ReactElement {
  return (
    <div className="section-gap">
      {blocks.map((block, index) => (
        <BlockView key={blockKey(block, index)} block={block} />
      ))}
    </div>
  );
}

/** Stable-ish key for a block based on its kind + first text content. */
function blockKey(block: HelpBlock, index: number): string {
  switch (block.kind) {
    case 'paragraph':
    case 'heading':
    case 'note':
      return `${block.kind}-${index}`;
    case 'terms':
    case 'steps':
    case 'tips':
      return `${block.kind}-${block.heading ?? index}`;
    default:
      return `block-${index}`;
  }
}

function BlockView({ block }: { block: HelpBlock }): ReactElement | null {
  switch (block.kind) {
    case 'paragraph':
      return <p className="body-small">{block.text}</p>;

    case 'heading':
      return <h3 className="heading-4">{block.text}</h3>;

    case 'note':
      return (
        <p className={cn(spacing.pad.sm, radius.default, 'bg-surface-hover body-small')}>
          {block.text}
        </p>
      );

    case 'terms':
      return (
        <section className="stack-sm">
          {block.heading ? <h3 className="heading-4">{block.heading}</h3> : null}
          <dl className="stack-sm">
            {block.items.map((item) => (
              <div
                key={item.term}
                className={cn(spacing.pad.sm, radius.default, 'bg-surface-hover')}
              >
                <dt className="label">{item.term}</dt>
                <dd className="caption mt-tight">{item.description}</dd>
              </div>
            ))}
          </dl>
        </section>
      );

    case 'steps':
      return (
        <section className="stack-sm">
          {block.heading ? <h3 className="heading-4">{block.heading}</h3> : null}
          {block.ordered ? (
            <ol className="stack-sm pl-indent list-decimal">
              {block.items.map((step, i) => (
                <li
                  key={step.title ?? `${i}-${step.description.slice(0, 24)}`}
                  className="body-small"
                >
                  {step.title ? <span className="label">{step.title}</span> : null}
                  {step.title ? (
                    <span className="caption"> — {step.description}</span>
                  ) : (
                    step.description
                  )}
                </li>
              ))}
            </ol>
          ) : (
            <div className="stack-sm">
              {block.items.map((step, i) => (
                <div
                  key={step.title ?? `${i}-${step.description.slice(0, 24)}`}
                  className={cn(spacing.pad.sm, radius.default, 'bg-surface-hover')}
                >
                  {step.title ? <p className="label">{step.title}</p> : null}
                  <p className={cn('caption', step.title ? 'mt-tight' : '')}>{step.description}</p>
                </div>
              ))}
            </div>
          )}
        </section>
      );

    case 'tips':
      return (
        <section className="stack-sm">
          {block.heading ? <h3 className="heading-4">{block.heading}</h3> : null}
          <ul className="stack-xs pl-indent list-disc">
            {block.items.map((tip) => (
              <li key={tip} className={cn(layout.flex.start, 'body-small')}>
                {tip}
              </li>
            ))}
          </ul>
        </section>
      );

    default:
      return null;
  }
}

export default HelpSectionBody;
