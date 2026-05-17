/**
 * Main entry point of @DFXswiss/bitbox-testkit.
 *
 * Consumers typically import from the namespaced subpaths:
 *
 *   import { FakePairedBitBox } from '@DFXswiss/bitbox-testkit/fake';
 *   import { Registry, subset } from '@DFXswiss/bitbox-testkit/quirks';
 *   import { scenarioRegressionUmlautEIP712 } from '@DFXswiss/bitbox-testkit/scenarios';
 *   import { detectNonAsciiInEIP712Literals } from '@DFXswiss/bitbox-testkit/guards';
 *
 * The default export re-exposes everything for convenience.
 */

export * from './errors.js';
export * as fake from './fake/index.js';
export * as quirks from './quirks/index.js';
export * as scenarios from './scenarios/index.js';
export * as guards from './guards/index.js';
