/**
 * Per-quirk callbacks: language-specific behaviour attached after loading
 * the JSON metadata. Mirrors /go/bitbox/quirks/callbacks.go.
 *
 * The TS callbacks are intentionally lighter than the Go ones:
 *   - `detect` operates on a list of source file paths and returns
 *     findings (no test runner involvement).
 *   - `scenario` returns a fake setup descriptor that the consumer's
 *     jest.mock invocation can install. Shape lives in /ts/src/fake.
 */

import type { Quirk, DetectFinding } from './types.js';
import {
  scenarioRegressionUmlautEIP712,
  scenarioErrInvalidInput,
  scenarioChannelHashEarly,
  scenarioPanicMidQuery,
  scenarioSlowResponse,
  scenarioDeviceDisconnect,
} from '../scenarios/index.js';
import {
  detectNonAsciiInEIP712Literals,
  detectBitBoxDedupOrder,
  detectNoHardcoded10sTimeout,
} from '../guards/index.js';

export function attachCallbacks(q: Quirk): void {
  // Default scenario for any wire-level firmware reject quirk.
  q.scenario = scenarioErrInvalidInput;

  switch (q.id) {
    case 'E1':
      q.detect = (paths) => detectNonAsciiInEIP712Literals(paths);
      q.scenario = scenarioRegressionUmlautEIP712;
      break;
    case 'P1':
      q.scenario = scenarioChannelHashEarly;
      break;
    case 'P2':
      q.detect = (paths) => detectBitBoxDedupOrder(paths);
      q.scenario = scenarioDeviceDisconnect;
      break;
    case 'A1':
      q.scenario = scenarioPanicMidQuery;
      break;
    case 'A2':
      q.detect = (paths) => detectNoHardcoded10sTimeout(paths);
      q.scenario = scenarioSlowResponse;
      break;
    // E7, C3, B5 — client-side enum/length validation; default scenario keeps wire-level mock,
    // but they typically surface as client-side parse errors rather than firmware 101s.
  }

  // Quiet unused-import guard if all the detect functions branched out:
  void DetectFindingHint;
}

// Re-export the type so consumers using `quirks/callbacks.js` don't need
// a separate import line.
type DetectFindingHint = DetectFinding;
const DetectFindingHint: DetectFinding | undefined = undefined;
