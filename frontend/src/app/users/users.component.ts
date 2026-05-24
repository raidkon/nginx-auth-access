import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import {
  AlertModule,
  ButtonModule,
  CalloutModule,
  CardModule,
  FormModule,
  GridModule,
  HeaderModule,
  SpinnerModule,
  TableModule,
} from '@coreui/angular';
import { IconModule } from '@coreui/icons-angular';

import { firstValueFrom } from 'rxjs';

import { ApiService, UserRow } from '../api.service';
import { TotpSetupDialogComponent } from './totp-setup-dialog.component';

@Component({
  selector: 'app-users',
  imports: [
    AlertModule,
    ButtonModule,
    CalloutModule,
    CardModule,
    FormModule,
    GridModule,
    HeaderModule,
    IconModule,
    SpinnerModule,
    TableModule,
    TotpSetupDialogComponent,
  ],
  templateUrl: './users.component.html',
  styleUrl: './users.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UsersComponent {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);

  readonly users = signal<UserRow[]>([]);
  readonly error = signal<string | null>(null);
  readonly bootstrap = signal(false);
  readonly totpWizardOpen = signal(false);
  readonly saving = signal(false);

  readonly newLogin = signal('');
  readonly newPassword = signal('');
  readonly newTotpSecret = signal('');
  readonly newAdmin = signal(false);

  readonly canAddUser = computed(
    () =>
      !this.saving() &&
      this.newLogin().trim().length > 0 &&
      this.newPassword().length > 0 &&
      this.newTotpSecret().trim().length > 0,
  );

  constructor() {
    void this.bootstrapPage();
  }

  patchNewLogin(value: string): void {
    this.newLogin.set(value);
  }

  patchNewPassword(value: string): void {
    this.newPassword.set(value);
  }

  patchNewAdmin(checked: boolean): void {
    this.newAdmin.set(checked);
  }

  private async bootstrapPage(): Promise<void> {
    await this.api.refreshSession();
    const s = this.api.session();
    if (!s?.ok) {
      await this.router.navigateByUrl('/login');
      return;
    }
    if (!s.admin) {
      this.error.set('Нет прав администратора');
      return;
    }
    this.bootstrap.set(!!s.bootstrap);
    await this.reload();
  }

  async reload(): Promise<void> {
    try {
      const r = await firstValueFrom(this.api.listUsers());
      this.users.set(r.users);
    } catch {
      this.error.set('Не удалось загрузить список');
    }
  }

  openTotpSetup(): void {
    this.error.set(null);
    const login = this.newLogin().trim();
    if (!login) {
      this.error.set('Сначала введите логин — он попадёт в подпись записи в приложении.');
      return;
    }
    this.totpWizardOpen.set(true);
  }

  applyTotpSecret(secret: string): void {
    this.newTotpSecret.set(secret);
    this.totpWizardOpen.set(false);
  }

  closeTotpWizard(): void {
    this.totpWizardOpen.set(false);
  }

  clearTotpSecret(): void {
    this.newTotpSecret.set('');
  }

  resetAddForm(): void {
    this.newLogin.set('');
    this.newPassword.set('');
    this.newTotpSecret.set('');
    this.newAdmin.set(false);
  }

  async add(): Promise<void> {
    this.error.set(null);
    if (!this.canAddUser()) {
      return;
    }
    this.saving.set(true);
    try {
      await firstValueFrom(
        this.api.addUser({
          login: this.newLogin().trim(),
          password: this.newPassword(),
          totp_secret: this.newTotpSecret().trim(),
          admin: this.newAdmin(),
        }),
      );
      this.resetAddForm();
      await this.api.refreshSession();
      this.bootstrap.set(false);
      await this.reload();
    } catch {
      this.error.set('Не удалось добавить пользователя');
    } finally {
      this.saving.set(false);
    }
  }

  async remove(login: string): Promise<void> {
    try {
      await firstValueFrom(this.api.deleteUser(login));
      await this.reload();
    } catch {
      this.error.set('Не удалось удалить');
    }
  }

  async logout(): Promise<void> {
    try {
      await firstValueFrom(this.api.logout());
    } catch {
      /* ignore */
    }
    this.api.session.set(null);
    await this.router.navigateByUrl('/login');
  }
}
