/** @jsxImportSource react */

import { Menu as MenuPrimitive } from '@videojs/react';
import { PlaybackRateRadioGroup } from '@videojs/react/ui/playback-rate-radio-group';

import { ButtonTooltip } from '@/components/videojs/ui/button-tooltip';
import { PlaybackRateButton } from '@/components/videojs/ui/playback-rate-button';
import { RadioItem } from '@/components/videojs/ui/radio-item';
import { cn } from '@/lib/resolve-class-name';

export interface AudioSettingsMenuProps extends Omit<MenuPrimitive.RootProps, 'children'> {
  className?: MenuPrimitive.TriggerProps['className'];
}

export function AudioSettingsMenu({ className, ...props }: AudioSettingsMenuProps = {}) {
  return (
    <MenuPrimitive.Root side="top" align="center" boundary="viewport" {...props}>
      <PlaybackRateRadioGroup.Root>
        <ButtonTooltip boundary="viewport" side="top">
          <MenuPrimitive.Trigger render={<PlaybackRateButton />} className={className} data-testid="player-speed" />
        </ButtonTooltip>
        <MenuPrimitive.Popup
          className={cn(
            'm-0 overflow-visible border-0 text-inherit',
            'media-transitioning:opacity-0 media-transitioning:blur-media-hidden-popup media-transitioning:scale-media-hidden-popup',
            'data-starting-style:transform-[translate(var(--media-popup-translate-x-distance,0),var(--media-popup-translate-y-distance,0))]',
            'data-ending-style:transform-none',
            'data-[side=top]:origin-bottom data-[side=bottom]:origin-top data-[side=left]:origin-right data-[side=right]:origin-left',
            'data-[side=top]:[--media-popup-translate-y-distance:var(--media-popup-translate-distance)]',
            'data-[side=bottom]:[--media-popup-translate-y-distance:calc(var(--media-popup-translate-distance)*-1)]',
            'data-[side=left]:[--media-popup-translate-x-distance:var(--media-popup-translate-distance)]',
            'data-[side=right]:[--media-popup-translate-x-distance:calc(var(--media-popup-translate-distance)*-1)]',
            'bg-media-popover text-media-popover-foreground surface-media after:surface-media-inset',
            'm-0 min-w-44 max-w-(--media-menu-available-width) overflow-hidden! rounded-media-popup border-0 p-1 [--media-popup-side-offset:var(--media-popover-side-offset)]',
            'max-h-[min(var(--media-menu-available-height,--spacing(56)),--spacing(56))] overscroll-none',
            'h-(--media-menu-height) w-(--media-menu-width)',
            'transition-media-popup media-transitioning:transition-media-popup',
            'min-w-0!'
          )}
        >
          <MenuPrimitive.Content
            className={cn(
              '[anchor-scope:--media-menu-item-highlight-anchor]',
              'media-anchored:before:anchor-media-highlight',
              'media-transitioning:before:hidden',
              'media-transitioning:[&_*]:before:hidden',
              'media-anchored:has-data-[highlighted=]:before:duration-0',
              'not-data-submenu:flex not-data-submenu:flex-col not-data-submenu:gap-0.5',
              'not-data-submenu:data-child-open:blur-media-hidden',
              'not-data-submenu:data-child-open:before:hidden',
              'data-submenu:media-transitioning:pointer-events-none data-submenu:media-transitioning:overflow-hidden',
              'data-submenu:media-transitioning:blur-media-hidden',
              'absolute overflow-auto overscroll-none outline-hidden',
              'transition-[translate,filter] duration-media-menu ease-out',
              'not-data-submenu:inset-x-1 not-data-submenu:top-1',
              'not-data-submenu:max-h-[calc(min(var(--media-menu-available-height,--spacing(56)),--spacing(56))-(--spacing(2)))]',
              'not-data-submenu:data-child-open:-translate-x-full',
              'not-data-submenu:data-child-open:rtl:translate-x-full',
              'data-submenu:inset-x-0 data-submenu:top-0 data-submenu:z-10 data-submenu:max-h-[inherit] data-submenu:p-1',
              'data-submenu:media-transitioning:translate-x-full data-submenu:media-transitioning:rtl:-translate-x-full'
            )}
          >
            <PlaybackRateRadioGroup.Options
              className={cn(
                'flex flex-col gap-0.5',
                '[anchor-scope:--media-menu-item-highlight-anchor]',
                'media-anchored:before:anchor-media-highlight',
                'media-transitioning:before:hidden',
                'media-transitioning:[&_*]:before:hidden',
                'media-anchored:has-data-[highlighted=]:before:duration-0'
              )}
              renderItem={(props, item) => (
                <RadioItem {...props}>
                  <span>{item.label}</span>
                </RadioItem>
              )}
            ></PlaybackRateRadioGroup.Options>
          </MenuPrimitive.Content>
        </MenuPrimitive.Popup>
      </PlaybackRateRadioGroup.Root>
    </MenuPrimitive.Root>
  );
}
