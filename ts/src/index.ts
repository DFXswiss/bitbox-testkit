/**
 * Main entry point of @joshuakrueger-dfx/bitbox-testkit.
 *
 * Consumers typically import from the namespaced subpaths:
 *
 *   import { FakePairedBitBox } from '@joshuakrueger-dfx/bitbox-testkit/fake';
 *   import { Registry, subset } from '@joshuakrueger-dfx/bitbox-testkit/quirks';
 *   import { scenarioRegressionUmlautEIP712 } from '@joshuakrueger-dfx/bitbox-testkit/scenarios';
 *   import { detectNonAsciiInEIP712Literals } from '@joshuakrueger-dfx/bitbox-testkit/guards';
 *
 * The default export re-exposes everything for convenience.
 */

export * from './errors.js';
export * as fake from './fake/index.js';
export * as quirks from './quirks/index.js';
export * as scenarios from './scenarios/index.js';
export * as guards from './guards/index.js';
