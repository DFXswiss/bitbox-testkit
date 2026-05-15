/**
 * Error types shared across the TS testkit.
 */

/** FirmwareError mirrors the wire-level error a real BitBox firmware emits. */
export class FirmwareError extends Error {
  readonly code: number;
  constructor(code: number, message: string) {
    super(message);
    this.name = 'FirmwareError';
    this.code = code;
  }
}

/** ErrInvalidInput101 — BitBox firmware "invalid input" response. */
export const ErrInvalidInput101 = new FirmwareError(101, 'firmware: invalid input (101)');

/** ErrUserAbort — user cancelled on-device. */
export const ErrUserAbort = new FirmwareError(104, 'firmware: user abort (104)');

/** ErrClosed — operation attempted after the fake was closed. */
export class ClosedError extends Error {
  constructor() {
    super('bitbox-testkit/fake: communication closed');
    this.name = 'ClosedError';
  }
}

/** ErrUnexpectedQuery — no fake handler matched an incoming call. */
export class UnexpectedQueryError extends Error {
  constructor(method: string) {
    super(`bitbox-testkit/fake: unexpected call to ${method}, no handler matched`);
    this.name = 'UnexpectedQueryError';
  }
}

/** ErrAwaitingUserConfirm — scenario waiting on signalConfirm(). */
export class AwaitingUserConfirmError extends Error {
  constructor() {
    super('bitbox-testkit/scenarios: awaiting user confirmation');
    this.name = 'AwaitingUserConfirmError';
  }
}
