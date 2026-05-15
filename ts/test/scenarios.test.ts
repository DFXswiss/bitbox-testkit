import { buildPairedBitBox } from '../src/fake/index.js';
import {
  scenarioRegressionUmlautEIP712,
  scenarioErrInvalidInput,
  scenarioPanicMidQuery,
  scenarioChannelHashEarly,
  scenarioDeviceDisconnect,
} from '../src/scenarios/index.js';
import { ErrInvalidInput101, AwaitingUserConfirmError } from '../src/errors.js';

describe('scenarioRegressionUmlautEIP712', () => {
  it('rejects non-ASCII payloads with ErrInvalidInput101', async () => {
    const proxy = buildPairedBitBox<{
      ethSignMessage: (chainId: bigint, keypath: number[], msg: Uint8Array) => Promise<unknown>;
    }>(scenarioRegressionUmlautEIP712());

    const utf8Umlaut = new TextEncoder().encode('Hëllo');
    await expect(proxy.ethSignMessage(1n, [], utf8Umlaut)).rejects.toBe(ErrInvalidInput101);
  });

  it('accepts pure ASCII payloads', async () => {
    const proxy = buildPairedBitBox<{
      ethSignMessage: (chainId: bigint, keypath: number[], msg: Uint8Array) => Promise<{ r: Uint8Array }>;
    }>(scenarioRegressionUmlautEIP712());

    const ascii = new TextEncoder().encode('Hello');
    const sig = await proxy.ethSignMessage(1n, [], ascii);
    expect(sig.r.length).toBe(32);
  });
});

describe('scenarioErrInvalidInput', () => {
  it('rejects every common method', async () => {
    const proxy = buildPairedBitBox<{ deviceInfo: () => Promise<unknown>; btcSignPSBT: () => Promise<unknown> }>(
      scenarioErrInvalidInput(),
    );
    await expect(proxy.deviceInfo()).rejects.toBe(ErrInvalidInput101);
    await expect(proxy.btcSignPSBT()).rejects.toBe(ErrInvalidInput101);
  });
});

describe('scenarioPanicMidQuery', () => {
  it('throws sync on n-th call, otherwise resolves', async () => {
    const proxy = buildPairedBitBox<{ deviceInfo: () => Promise<unknown> }>(scenarioPanicMidQuery(2, 'boom'));
    await expect(proxy.deviceInfo()).resolves.toBeUndefined();
    await expect(proxy.deviceInfo()).rejects.toBe('boom');
  });
});

describe('scenarioChannelHashEarly', () => {
  it('hash repeats then blocks until signalConfirm', async () => {
    const setup = scenarioChannelHashEarly(2);
    const proxy = buildPairedBitBox<{ deviceInfo: () => Promise<unknown> }>(setup);
    // 2 hash repeats first
    await expect(proxy.deviceInfo()).resolves.toEqual({ channelHash: expect.any(Uint8Array) });
    await expect(proxy.deviceInfo()).resolves.toEqual({ channelHash: expect.any(Uint8Array) });
    // then blocks
    await expect(proxy.deviceInfo()).rejects.toBeInstanceOf(AwaitingUserConfirmError);
    setup.signalConfirm();
    await expect(proxy.deviceInfo()).resolves.toBeUndefined();
  });
});

describe('scenarioDeviceDisconnect', () => {
  it('rejects after N successful calls', async () => {
    const proxy = buildPairedBitBox<{ deviceInfo: () => Promise<unknown> }>(scenarioDeviceDisconnect(1));
    await expect(proxy.deviceInfo()).resolves.toBeUndefined();
    await expect(proxy.deviceInfo()).rejects.toThrow(/user abort|104/i);
  });
});
