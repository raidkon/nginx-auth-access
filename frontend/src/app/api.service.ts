import { HttpClient } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export interface LoginRequest {
  login: string;
  password: string;
  totp: string;
  period: string;
}

export interface SessionResponse {
  ok: boolean;
  login?: string;
  admin?: boolean;
  bootstrap?: boolean;
}

export interface UserRow {
  login: string;
  admin: boolean;
  /** Секрет в списке не отдаётся; только факт настройки TOTP. */
  has_totp: boolean;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);

  readonly session = signal<SessionResponse | null>(null);
  readonly isAuthenticated = computed(() => this.session()?.ok === true);
  readonly isAdmin = computed(() => this.session()?.admin === true);

  async refreshSession(): Promise<void> {
    const s = await firstValueFrom(this.http.get<SessionResponse>('/api/v1/session'));
    this.session.set(s);
  }

  login(body: LoginRequest) {
    return this.http.post<{ ok: boolean; login?: string; bootstrap?: boolean }>('/api/v1/login', body);
  }

  logout() {
    return this.http.post<{ ok: boolean }>('/api/v1/logout', {});
  }

  listUsers() {
    return this.http.get<{ users: UserRow[] }>('/api/v1/users');
  }

  addUser(body: { login: string; password: string; totp_secret: string; admin: boolean }) {
    return this.http.post<{ ok: boolean }>('/api/v1/users', body);
  }

  deleteUser(login: string) {
    return this.http.delete<{ ok: boolean }>(`/api/v1/users/${encodeURIComponent(login)}`);
  }
}
