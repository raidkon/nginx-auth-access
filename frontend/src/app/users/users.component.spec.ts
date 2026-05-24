import { provideHttpClient } from '@angular/common/http';
import {
    HttpTestingController,
    provideHttpClientTesting,
    TestRequest,
} from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { provideRouter, Router } from '@angular/router';

import { ApiService } from '../api.service';
import { UsersComponent } from './users.component';

interface SessionPayload {
    ok: boolean;
    admin?: boolean;
    bootstrap?: boolean;
    login?: string;
}

describe('UsersComponent', () => {
    let fixture: ComponentFixture<UsersComponent>;
    let component: UsersComponent;
    let httpMock: HttpTestingController;
    let router: Router;

    async function expectOneEventually(url: string, method = 'GET'): Promise<TestRequest> {
        for (let i = 0; i < 50; i++) {
            try {
                return httpMock.expectOne((req) => req.method === method && req.url === url);
            } catch {
                await new Promise((resolve) => setTimeout(resolve, 0));
            }
        }
        throw new Error(`Timed out waiting for ${method} ${url}`);
    }

    async function flushBootstrap(session: SessionPayload, users: object[] = []): Promise<void> {
        const sessionReq = httpMock.expectOne('/api/v1/session');
        sessionReq.flush(session);
        await fixture.whenStable();

        if (session.ok && session.admin) {
            const usersReq = await expectOneEventually('/api/v1/users');
            usersReq.flush({ users });
            await fixture.whenStable();
        }

        fixture.detectChanges();
    }

    async function createAdmin(session: SessionPayload = { ok: true, admin: true, bootstrap: false }, users: object[] = []): Promise<void> {
        fixture = TestBed.createComponent(UsersComponent);
        component = fixture.componentInstance;
        fixture.detectChanges();
        await flushBootstrap(session, users);
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [UsersComponent],
            providers: [
                ApiService,
                provideHttpClient(),
                provideHttpClientTesting(),
                provideRouter([]),
                provideNoopAnimations(),
            ],
        }).compileComponents();

        httpMock = TestBed.inject(HttpTestingController);
        router = TestBed.inject(Router);
    });

    afterEach(() => {
        httpMock.verify();
        TestBed.resetTestingModule();
    });

    it('redirects to login when session is not ok', async () => {
        vi.spyOn(router, 'navigateByUrl').mockReturnValue(Promise.resolve(true));
        await createAdmin({ ok: false });
        expect(router.navigateByUrl).toHaveBeenCalledWith('/login');
    });

    it('shows error when user is not admin', async () => {
        await createAdmin({ ok: true, admin: false, login: 'bob' });
        expect(component.error()).toBe('Нет прав администратора');
    });

    it('loads users for admin session', async () => {
        await createAdmin({ ok: true, admin: true, bootstrap: false }, [{ login: 'alice', admin: true, has_totp: true }]);
        expect(component.users().length).toBe(1);
        expect(component.users()[0].login).toBe('alice');
    });

    it('sets bootstrap flag from session', async () => {
        await createAdmin({ ok: true, admin: true, bootstrap: true }, []);
        expect(component.bootstrap()).toBe(true);
    });

    it('reload sets error on failure', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        const reloadPromise = component.reload();
        httpMock.expectOne('/api/v1/users').flush({}, { status: 500, statusText: 'Error' });
        await reloadPromise;
        expect(component.error()).toBe('Не удалось загрузить список');
    });

    it('openTotpSetup requires login', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.openTotpSetup();
        expect(component.error()).toContain('логин');
        expect(component.totpWizardOpen()).toBe(false);
    });

    it('openTotpSetup opens wizard when login is set', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.patchNewLogin('newuser');
        component.openTotpSetup();
        expect(component.totpWizardOpen()).toBe(true);
    });

    it('applyTotpSecret stores secret and closes wizard', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.totpWizardOpen.set(true);
        component.applyTotpSecret('JBSWY3DPEHPK3PXP');
        expect(component.newTotpSecret()).toBe('JBSWY3DPEHPK3PXP');
        expect(component.totpWizardOpen()).toBe(false);
    });

    it('closeTotpWizard closes dialog', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.totpWizardOpen.set(true);
        component.closeTotpWizard();
        expect(component.totpWizardOpen()).toBe(false);
    });

    it('clearTotpSecret clears secret field', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.applyTotpSecret('SECRET');
        component.clearTotpSecret();
        expect(component.newTotpSecret()).toBe('');
    });

    it('patchNewAdmin updates admin flag', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.patchNewAdmin(true);
        expect(component.newAdmin()).toBe(true);
    });

    it('resetAddForm clears add-user fields', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.patchNewLogin('x');
        component.patchNewPassword('y');
        component.applyTotpSecret('SECRET');
        component.patchNewAdmin(true);
        component.resetAddForm();
        expect(component.newLogin()).toBe('');
        expect(component.newPassword()).toBe('');
        expect(component.newTotpSecret()).toBe('');
        expect(component.newAdmin()).toBe(false);
    });

    it('add does nothing when form is incomplete', async () => {
        await createAdmin({ ok: true, admin: true, bootstrap: true }, []);
        await component.add();
        httpMock.expectNone('/api/v1/users');
    });

    it('add posts user and reloads list', async () => {
        await createAdmin({ ok: true, admin: true, bootstrap: true }, []);

        component.patchNewLogin('bob');
        component.patchNewPassword('pass');
        component.applyTotpSecret('JBSWY3DPEHPK3PXP');
        component.patchNewAdmin(true);
        expect(component.canAddUser()).toBe(true);

        const addPromise = component.add();
        await fixture.whenStable();

        const postReq = httpMock.expectOne((r) => r.method === 'POST' && r.url === '/api/v1/users');
        expect(postReq.request.method).toBe('POST');
        expect(postReq.request.body).toEqual({
            login: 'bob',
            password: 'pass',
            totp_secret: 'JBSWY3DPEHPK3PXP',
            admin: true,
        });
        postReq.flush({ ok: true });

        await fixture.whenStable();
        httpMock.expectOne('/api/v1/session').flush({ ok: true, admin: true, bootstrap: false });
        await new Promise((resolve) => setTimeout(resolve, 0));

        const reloadReq = httpMock.expectOne((r) => r.method === 'GET' && r.url === '/api/v1/users');
        reloadReq.flush({ users: [{ login: 'bob', admin: true, has_totp: true }] });

        await addPromise;

        expect(component.bootstrap()).toBe(false);
        expect(component.newLogin()).toBe('');
    });

    it('add sets error on failure', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        component.patchNewLogin('bob');
        component.patchNewPassword('pass');
        component.applyTotpSecret('JBSWY3DPEHPK3PXP');

        const addPromise = component.add();
        httpMock.expectOne('/api/v1/users').flush({}, { status: 400, statusText: 'Bad' });
        await addPromise;

        expect(component.error()).toBe('Не удалось добавить пользователя');
    });

    it('remove deletes user and reloads', async () => {
        await createAdmin({ ok: true, admin: true }, [{ login: 'bob', admin: false, has_totp: true }]);

        const removePromise = component.remove('bob');
        httpMock.expectOne('/api/v1/users/bob').flush({ ok: true });
        await fixture.whenStable();
        httpMock.expectOne('/api/v1/users').flush({ users: [] });
        await removePromise;

        expect(component.users().length).toBe(0);
    });

    it('remove sets error on failure', async () => {
        await createAdmin({ ok: true, admin: true }, []);
        const removePromise = component.remove('bob');
        httpMock.expectOne('/api/v1/users/bob').flush({}, { status: 500, statusText: 'Err' });
        await removePromise;
        expect(component.error()).toBe('Не удалось удалить');
    });

    it('logout clears session and navigates to login', async () => {
        vi.spyOn(router, 'navigateByUrl').mockReturnValue(Promise.resolve(true));
        const api = TestBed.inject(ApiService);
        await createAdmin({ ok: true, admin: true }, []);

        const logoutPromise = component.logout();
        httpMock.expectOne('/api/v1/logout').flush({ ok: true });
        await logoutPromise;

        expect(api.session()).toBeNull();
        expect(router.navigateByUrl).toHaveBeenCalledWith('/login');
    });
});
