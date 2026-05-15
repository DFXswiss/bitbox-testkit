/**
 * Type shapes for the in-memory fake of bitbox-api.
 *
 * The real `bitbox-api` ships `BitBox`, `PairingBitBox`, `PairedBitBox`
 * classes. Our fakes implement the subset the testkit cares about plus
 * scripting hooks for tests.
 */

/** Async function returning the mocked response for one bitbox-api call. */
export type Handler<TArgs extends readonly unknown[] = unknown[], TResult = unknown> = (
  ...args: TArgs
) => Promise<TResult>;

/** Setup descriptor returned by Scenario factories. Consumers pass this to
 *  `installFakePairedBitBox()` (or apply it manually in jest.mock setups). */
export interface FakeSetup {
  /** Map of method name → handler. Methods not present throw UnexpectedQueryError. */
  methods: Record<string, Handler>;
  /** Optional metadata for diagnostics. */
  description?: string;
}
