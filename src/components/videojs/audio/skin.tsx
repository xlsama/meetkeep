/** @jsxImportSource react */

'use client';

import '../styles/audio/base.css';
import type { ComponentProps, ReactNode } from 'react';

import { Container } from '@/components/videojs/ui/container';
import { StatusAnnouncer } from '@/components/videojs/ui/status-announcer';
import { cn } from '@/lib/resolve-class-name';

import { PlaybackHotkeys } from './behaviors/playback-hotkeys';
import { AudioErrorDialog } from './dialogs/error-dialog';
import { DefaultAudioControls } from './layout/controls';

export interface AudioSkinProps extends Omit<NonNullable<ComponentProps<typeof Container>>, 'children'> {
  children?: ReactNode;
  markers?: ReactNode;
}

export function AudioSkin({ children, className, markers, ...props }: AudioSkinProps = {}) {
  return (
    <Container
      className={cn('h-auto! overflow-visible! bg-transparent! after:hidden', className)}
      data-theme="default"
      data-preset="audio"
      {...props}
    >
      {children}
      <AudioErrorDialog />
      <DefaultAudioControls markers={markers} />
      <PlaybackHotkeys />
      <StatusAnnouncer />
    </Container>
  );
}
