/**
 * Types mirroring /go/bitbox/quirks/registry.go. Keep field names in sync
 * with quirks.json schema; see /quirks/SCHEMA.md for canonical semantics.
 */

export type Category =
  | 'eth'
  | 'btc'
  | 'cardano'
  | 'mnemonic'
  | 'protocol'
  | 'app';

export type Severity = 'hint' | 'warning' | 'critical';

export interface FirmwareRange {
  /** inclusive lower bound; empty means "from the beginning" */
  min: string;
  /** exclusive upper bound; empty means "forever" */
  max: string;
}

/**
 * One documented BitBox firmware constraint.
 */
export interface Quirk {
  readonly id: string;
  readonly name: string;
  readonly category: Category;
  readonly severity: Severity;
  readonly description: string;
  readonly source: string;
  readonly firmware: FirmwareRange;
  /** regex (as a string) matching Jest output that could indicate this quirk */
  readonly matchRegex?: string;

  /**
   * Source-level static check. Consumer passes a glob of files to scan.
   * May be undefined for quirks that have no statically detectable pattern.
   */
  detect?: (sourcePaths: string[]) => readonly DetectFinding[];

  /**
   * Scenario factory: returns a mock setup function. Pass it to
   * `installMocks()` (or call inside `jest.mock()`) to make the bitbox-api
   * surface return the documented firmware response.
   *
   * The returned function is wallet-API-specific (it produces a fake
   * PairedBitBox), so its concrete shape lives in /ts/src/fake.
   */
  scenario?: () => unknown;
}

export interface DetectFinding {
  readonly file: string;
  readonly line: number;
  readonly snippet: string;
  readonly reason: string;
}

/**
 * Filter applied to the global registry.
 */
export interface Filter {
  category?: Category;
  /** Lower bound on severity ranking: hint < warning < critical. */
  minSeverity?: Severity;
  /** Firmware version (e.g. "9.23.0"). When provided, only quirks whose
   *  FirmwareRange applies to this version are returned. */
  firmware?: string;
}
