import { routes } from './app.routes';

describe('app routes', () => {
  it('redirects empty path to login', () => {
    const root = routes.find((r) => r.path === '');
    expect(root?.redirectTo).toBe('login');
    expect(root?.pathMatch).toBe('full');
  });

  it('defines login route', () => {
    const login = routes.find((r) => r.path === 'login');
    expect(login?.loadComponent).toBeDefined();
  });

  it('defines users admin route', () => {
    const users = routes.find((r) => r.path === 'app/users');
    expect(users?.loadComponent).toBeDefined();
  });

  it('redirects unknown paths to login', () => {
    const fallback = routes.find((r) => r.path === '**');
    expect(fallback?.redirectTo).toBe('login');
  });
});
