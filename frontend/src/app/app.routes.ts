import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'login' },
  {
    path: 'login',
    loadComponent: () => import('./login/login.component').then((m) => m.LoginComponent),
  },
  {
    path: 'app/users',
    loadComponent: () => import('./users/users.component').then((m) => m.UsersComponent),
  },
  { path: '**', redirectTo: 'login' },
];
