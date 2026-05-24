import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  input,
  output,
  signal,
} from '@angular/core';
import {
  AlertModule,
  ButtonModule,
  FormModule,
  ModalModule,
} from '@coreui/angular';
import { Secret, TOTP } from 'otpauth';
import QRCode from 'qrcode';

type Step = 'choose' | 'qr' | 'manual';

@Component({
  selector: 'app-totp-setup-dialog',
  imports: [AlertModule, ButtonModule, FormModule, ModalModule],
  templateUrl: './totp-setup-dialog.component.html',
  styleUrl: './totp-setup-dialog.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TotpSetupDialogComponent {
  /** Логин нового пользователя — подпись в otpauth URI. */
  readonly login = input.required<string>();

  readonly secretApplied = output<string>();
  readonly cancelled = output<void>();

  readonly modalVisible = signal(true);
  readonly step = signal<Step>('choose');
  readonly qrDataUrl = signal<string | null>(null);
  readonly verifyError = signal<string | null>(null);
  readonly code = signal('');
  readonly secretBase32 = signal('');

  readonly codeValid = computed(() => /^\d{6}$/.test(this.code().trim()));

  private skipCancelOnClose = false;
  private secret!: Secret;
  private totp!: TOTP;

  constructor() {
    effect(() => {
      this.login();
      this.bootstrapCrypto();
    });
  }

  patchCode(value: string): void {
    this.code.set(value);
  }

  private bootstrapCrypto(): void {
    const label = this.login().trim() || 'user';
    this.secret = new Secret({ size: 20 });
    this.totp = new TOTP({
      issuer: 'home-smart access',
      label,
      algorithm: 'SHA1',
      digits: 6,
      period: 30,
      secret: this.secret,
      issuerInLabel: true,
    });
    this.secretBase32.set(this.secret.base32.toUpperCase());
    this.step.set('choose');
    this.qrDataUrl.set(null);
    this.verifyError.set(null);
    this.code.set('');
  }

  async chooseQr(): Promise<void> {
    this.verifyError.set(null);
    this.code.set('');
    this.step.set('qr');
    const uri = this.totp.toString();
    const url = await QRCode.toDataURL(uri, { width: 260, margin: 2, errorCorrectionLevel: 'M' });
    this.qrDataUrl.set(url);
  }

  chooseManual(): void {
    this.verifyError.set(null);
    this.code.set('');
    this.step.set('manual');
  }

  backToChoose(): void {
    this.verifyError.set(null);
    this.code.set('');
    this.step.set('choose');
    this.qrDataUrl.set(null);
  }

  confirm(): void {
    this.verifyError.set(null);
    const token = this.code().trim();
    if (!this.codeValid()) {
      this.verifyError.set('Введите 6-значный код из приложения.');
      return;
    }
    const delta = this.totp.validate({ token, window: 1 });
    if (delta === null) {
      this.verifyError.set('Код неверный или устарел. Проверьте время на телефоне и попробуйте снова.');
      return;
    }
    this.skipCancelOnClose = true;
    this.secretApplied.emit(this.secretBase32());
    this.modalVisible.set(false);
  }

  cancel(): void {
    this.modalVisible.set(false);
  }

  onModalVisibleChange(visible: boolean): void {
    this.modalVisible.set(visible);
    if (!visible) {
      if (this.skipCancelOnClose) {
        this.skipCancelOnClose = false;
        return;
      }
      this.cancelled.emit();
    }
  }
}
