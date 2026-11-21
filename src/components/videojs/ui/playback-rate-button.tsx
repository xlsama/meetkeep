/** @jsxImportSource react */

'use client';

import '../styles/base.css';
import '../styles/audio/theme.css';
import '../styles/video/captions.css';
import '../styles/video/theme.css';
import { PlaybackRateButton as PlaybackRateButtonPrimitive } from '@videojs/react';

import { Button } from '@/components/videojs/ui/button';
import { cn, resolveClassName } from '@/lib/resolve-class-name';

export type PlaybackRateButtonProps = Omit<PlaybackRateButtonPrimitive.Props, 'children'>;

export function PlaybackRateButton({ className, ...props }: PlaybackRateButtonProps = {}) {
  return (
    <PlaybackRateButtonPrimitive
      render={<Button />}
      className={(state) =>
        cn(
          'tabular-nums not-supports-[text-fit:shrink]:text-[length:min(1em,calc(4em/(var(--media-playback-rate-length,3)_+_1)))]',
          "after:content-[attr(data-rate)_'×'] after:w-[80%] after:min-w-0 after:whitespace-nowrap after:text-center after:[text-fit:shrink]",
          resolveClassName(className, state)
        )
      }
      {...props}
    />
  );
}
