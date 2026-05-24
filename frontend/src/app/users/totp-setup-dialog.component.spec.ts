import { TestBed } from '@angular/core/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { ComponentFixture } from '@angular/core/testing';
import { Secret, TOTP } from 'otpauth';

import { TotpSetupDialogComponent } from './totp-setup-dialog.component';

describe('TotpSetupDialogComponent', () => {
    let fixture: ComponentFixture<TotpSetupDialogComponent>;
    let component: TotpSetupDialogComponent;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TotpSetupDialogComponent],
            providers: [provideNoopAnimations()],
        }).compileComponents();

        fixture = TestBed.createComponent(TotpSetupDialogComponent);
        component = fixture.componentInstance;
        fixture.componentRef.setInput('login', 'testuser');
        fixture.detectChanges();
    });

    it('should create and bootstrap secret', () => {
        expect(component).toBeTruthy();
        expect(component.secretBase32().length).toBeGreaterThan(0);
        expect(component.step()).toBe('choose');
    });

    it('codeValid accepts 6 digits only', () => {
        component.patchCode('12345');
        expect(component.codeValid()).toBe(false);
        component.patchCode('123456');
        expect(component.codeValid()).toBe(true);
        component.patchCode('1234567');
        expect(component.codeValid()).toBe(false);
    });

    it('chooseManual switches to manual step', () => {
        component.chooseManual();
        expect(component.step()).toBe('manual');
        expect(component.qrDataUrl()).toBeNull();
    });

    it('chooseQr generates data URL', async () => {
        await component.chooseQr();
        expect(component.step()).toBe('qr');
        expect(component.qrDataUrl()).toMatch(/^data:image\/png;base64,/);
    });

    it('backToChoose resets wizard state', async () => {
        await component.chooseQr();
        component.patchCode('123456');
        component.backToChoose();
        expect(component.step()).toBe('choose');
        expect(component.qrDataUrl()).toBeNull();
        expect(component.code()).toBe('');
    });

    it('confirm rejects invalid code length', () => {
        component.patchCode('12');
        component.confirm();
        expect(component.verifyError()).toContain('6-значный');
    });

    it('confirm rejects wrong TOTP token', () => {
        component.patchCode('000000');
        component.confirm();
        expect(component.verifyError()).toContain('неверный');
    });

    it('confirm emits secret on valid TOTP', () => {
        const base32 = component.secretBase32();
        const totp = new TOTP({
            secret: Secret.fromBase32(base32),
            algorithm: 'SHA1',
            digits: 6,
            period: 30,
        });
        const token = totp.generate();

        vi.spyOn(component.secretApplied, 'emit');
        component.patchCode(token);
        component.confirm();

        expect(component.secretApplied.emit).toHaveBeenCalledWith(base32);
        expect(component.modalVisible()).toBe(false);
    });

    it('cancel closes modal', () => {
        component.cancel();
        expect(component.modalVisible()).toBe(false);
    });

    it('onModalVisibleChange emits cancelled when closed without confirm', () => {
        vi.spyOn(component.cancelled, 'emit');
        component.onModalVisibleChange(false);
        expect(component.cancelled.emit).toHaveBeenCalled();
    });

    it('onModalVisibleChange skips cancelled after successful confirm', () => {
        const base32 = component.secretBase32();
        const totp = new TOTP({
            secret: Secret.fromBase32(base32),
            algorithm: 'SHA1',
            digits: 6,
            period: 30,
        });

        vi.spyOn(component.cancelled, 'emit');
        component.patchCode(totp.generate());
        component.confirm();
        component.onModalVisibleChange(false);
        expect(component.cancelled.emit).not.toHaveBeenCalled();
    });

    it('re-bootstrap secret when login input changes', () => {
        const first = component.secretBase32();
        fixture.componentRef.setInput('login', 'other');
        fixture.detectChanges();
        expect(component.secretBase32()).not.toBe(first);
    });
});
