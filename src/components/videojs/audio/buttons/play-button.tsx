/** @jsxImportSource react */

import type { ComponentProps } from 'react';

import { BufferingIndicator } from '@/components/videojs/ui/buffering-indicator';
import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { PlayButton } from '@/components/videojs/ui/play-button';
import { cn } from '@/lib/resolve-class-name';

export type AudioPlayButtonProps = Omit<ComponentProps<'div'>, 'children'>;

export function AudioPlayButton({ className, ...props }: AudioPlayButtonProps = {}) {
  return (
    <div className={cn('relative inline-flex', className)} {...props}>
      <BufferingIndicator
        className={cn('z-20 rounded-media-control text-inherit! before:hidden', 'data-visible:bg-media-controls')}
      />
      <ButtonTooltip boundary="viewport" side="top">
        <PlayButton data-testid="player-toggle" />
      </ButtonTooltip>
    </div>
  );
}
