import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { ApiService } from './api.service';

describe('ApiService', () => {
    let service: ApiService;
    let httpMock: HttpTestingController;

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [ApiService, provideHttpClient(), provideHttpClientTesting()],
        });
        service = TestBed.inject(ApiService);
        httpMock = TestBed.inject(HttpTestingController);
    });

    afterEach(() => {
        httpMock.verify();
    });

    it('should be created', () => {
        expect(service).toBeTruthy();
    });

    it('refreshSession loads session into signal', async () => {
        const promise = service.refreshSession();
        const req = httpMock.expectOne('/api/v1/session');
        expect(req.request.method).toBe('GET');
        req.flush({ ok: true, login: 'alice', admin: true });
        await promise;
        expect(service.session()).toEqual({ ok: true, login: 'alice', admin: true });
        expect(service.isAuthenticated()).toBe(true);
        expect(service.isAdmin()).toBe(true);
    });

    it('isAuthenticated and isAdmin are false without session', () => {
        expect(service.isAuthenticated()).toBe(false);
        expect(service.isAdmin()).toBe(false);
    });

    it('login posts credentials', () => {
        const body = { login: 'u', password: 'p', totp: '123456', period: '1h' };
        service.login(body).subscribe();
        const req = httpMock.expectOne('/api/v1/login');
        expect(req.request.method).toBe('POST');
        expect(req.request.body).toEqual(body);
        req.flush({ ok: true, login: 'u' });
    });

    it('logout posts empty body', () => {
        service.logout().subscribe();
        const req = httpMock.expectOne('/api/v1/logout');
        expect(req.request.method).toBe('POST');
        expect(req.request.body).toEqual({});
        req.flush({ ok: true });
    });

    it('listUsers fetches users', () => {
        service.listUsers().subscribe((r) => {
            expect(r.users.length).toBe(1);
        });
        const req = httpMock.expectOne('/api/v1/users');
        expect(req.request.method).toBe('GET');
        req.flush({ users: [{ login: 'bob', admin: false, has_totp: true }] });
    });

    it('addUser posts new user', () => {
        const body = {
            login: 'bob',
            password: 'secret',
            totp_secret: 'JBSWY3DPEHPK3PXP',
            admin: true,
        };
        service.addUser(body).subscribe();
        const req = httpMock.expectOne('/api/v1/users');
        expect(req.request.method).toBe('POST');
        expect(req.request.body).toEqual(body);
        req.flush({ ok: true });
    });

    it('deleteUser encodes login in URL', () => {
        service.deleteUser('user@host').subscribe();
        const req = httpMock.expectOne('/api/v1/users/user%40host');
        expect(req.request.method).toBe('DELETE');
        req.flush({ ok: true });
    });
});
