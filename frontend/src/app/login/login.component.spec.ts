import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { provideRouter, Router } from '@angular/router';

import { ApiService } from '../api.service';
import { LoginComponent } from './login.component';

describe('LoginComponent', () => {
    let fixture: ComponentFixture<LoginComponent>;
    let component: LoginComponent;
    let httpMock: HttpTestingController;
    let router: Router;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [LoginComponent],
            providers: [
                ApiService,
                provideHttpClient(),
                provideHttpClientTesting(),
                provideRouter([]),
                provideNoopAnimations(),
            ],
        }).compileComponents();

        fixture = TestBed.createComponent(LoginComponent);
        component = fixture.componentInstance;
        httpMock = TestBed.inject(HttpTestingController);
        router = TestBed.inject(Router);
        fixture.detectChanges();
    });

    afterEach(() => {
        httpMock.verify();
    });

    it('should create', () => {
        expect(component).toBeTruthy();
    });

    it('canSubmit is false when fields are empty', () => {
        expect(component.canSubmit()).toBe(false);
    });

    it('canSubmit is true when all fields are filled', () => {
        component.patchLogin('alice');
        component.patchPassword('secret');
        component.patchTotp('123456');
        expect(component.canSubmit()).toBe(true);
    });

    it('patchPeriod updates period signal', () => {
        component.patchPeriod('8h');
        expect(component.period()).toBe('8h');
    });

    it('submit sets error when fields are incomplete', async () => {
        await component.submit();
        expect(component.error()).toBe('Заполните все поля');
        httpMock.expectNone('/api/v1/login');
    });

    it('submit posts login and navigates on success', async () => {
        vi.spyOn(router, 'navigateByUrl').mockReturnValue(Promise.resolve(true));

        component.patchLogin(' alice ');
        component.patchPassword('secret');
        component.patchTotp(' 123456 ');
        component.patchPeriod('1h');

        const submitPromise = component.submit();

        const loginReq = httpMock.expectOne('/api/v1/login');
        expect(loginReq.request.method).toBe('POST');
        expect(loginReq.request.body).toEqual({
            login: 'alice',
            password: 'secret',
            totp: '123456',
            period: '1h',
        });
        loginReq.flush({ ok: true, login: 'alice' });

        await fixture.whenStable();
        const sessionReq = httpMock.expectOne('/api/v1/session');
        sessionReq.flush({ ok: true, login: 'alice', admin: true });

        await submitPromise;

        expect(router.navigateByUrl).toHaveBeenCalledWith('/app/users');
        expect(component.busy()).toBe(false);
    });

    it('submit sets error on server failure', async () => {
        component.patchLogin('alice');
        component.patchPassword('secret');
        component.patchTotp('123456');

        const submitPromise = component.submit();
        const loginReq = httpMock.expectOne('/api/v1/login');
        loginReq.flush({ error: 'bad' }, { status: 401, statusText: 'Unauthorized' });

        await submitPromise;

        expect(component.error()).toBe('Неверные данные или ошибка сервера');
        expect(component.busy()).toBe(false);
    });

    it('form submit triggers POST login (not native GET navigation)', async () => {
        vi.spyOn(router, 'navigateByUrl').mockReturnValue(Promise.resolve(true));

        component.patchLogin('alice');
        component.patchPassword('secret');
        component.patchTotp('123456');
        fixture.detectChanges();

        const form = fixture.nativeElement.querySelector('form') as HTMLFormElement;
        form.requestSubmit();

        await fixture.whenStable();

        const loginReq = httpMock.expectOne('/api/v1/login');
        expect(loginReq.request.method).toBe('POST');
        loginReq.flush({ ok: true });

        await fixture.whenStable();
        httpMock.expectOne('/api/v1/session').flush({ ok: true, admin: true });

        await fixture.whenStable();
    });
});
