import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import {
  AlertModule,
  ButtonModule,
  CalloutModule,
  CardModule,
  FormModule,
  SpinnerModule,
} from '@coreui/angular';
import { IconModule } from '@coreui/icons-angular';

import { firstValueFrom } from 'rxjs';

import { ApiService } from '../api.service';

const PERIODS = [
  { v: '30m', l: '30 мин' },
  { v: '1h', l: '1 ч' },
  { v: '3h', l: '3 ч' },
  { v: '8h', l: '8 ч' },
  { v: '24h', l: '24 ч' },
] as const;

@Component({
  selector: 'app-login',
  imports: [
    AlertModule,
    ButtonModule,
    CalloutModule,
    CardModule,
    FormModule,
    FormsModule,
    IconModule,
    SpinnerModule,
  ],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginComponent {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);

  readonly periods = PERIODS;

  readonly login = signal('');
  readonly password = signal('');
  readonly totp = signal('');
  readonly period = signal<string>('30m');
  readonly error = signal<string | null>(null);
  readonly busy = signal(false);

  readonly canSubmit = computed(
    () =>
      !this.busy() &&
      this.login().trim().length > 0 &&
      this.password().length > 0 &&
      this.totp().trim().length > 0,
  );

  patchLogin(value: string): void {
    this.login.set(value);
  }

  patchPassword(value: string): void {
    this.password.set(value);
  }

  patchTotp(value: string): void {
    this.totp.set(value);
  }

  patchPeriod(value: string): void {
    this.period.set(value);
  }

  async submit(): Promise<void> {
    this.error.set(null);
    if (!this.canSubmit()) {
      this.error.set('Заполните все поля');
      return;
    }
    this.busy.set(true);
    try {
      await firstValueFrom(
        this.api.login({
          login: this.login().trim(),
          password: this.password(),
          totp: this.totp().trim(),
          period: this.period(),
        }),
      );
      await this.api.refreshSession();
      await this.router.navigateByUrl('/app/users');
    } catch {
      this.error.set('Неверные данные или ошибка сервера');
    } finally {
      this.busy.set(false);
    }
  }
}
