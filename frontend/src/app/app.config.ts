import {
  ApplicationConfig,
  importProvidersFrom,
  inject,
  provideAppInitializer,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideHttpClient, withFetch } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { freeSet } from '@coreui/icons';
import { IconSetModule, IconSetService } from '@coreui/icons-angular';

import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    provideAnimationsAsync(),
    importProvidersFrom(IconSetModule),
    provideAppInitializer(() => {
      const iconSet = inject(IconSetService);
      iconSet.icons = { ...freeSet };
    }),
    provideHttpClient(withFetch()),
    provideRouter(routes),
  ],
};
