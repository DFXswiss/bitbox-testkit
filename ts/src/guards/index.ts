/**
 * Source-level regression guards for TypeScript/JavaScript consumers.
 * Mirrors /go/core/guards but operates on .ts/.tsx/.js source.
 *
 * Each guard reads files from disk, applies a regex, and returns findings.
 * Consumers wire them into their own test suite — e.g. Jest test that
 * fails when findings are non-empty.
 */

import * as fs from 'node:fs';
import * as path from 'node:path';
import type { DetectFinding } from '../quirks/types.js';

/**
 * Walk a list of source paths, returning every line that matches `pattern`.
 * Skips files we can't read (returns no findings for them).
 */
function scan(paths: string[], pattern: RegExp, reason: string): DetectFinding[] {
  const out: DetectFinding[] = [];
  for (const p of paths) {
    let content: string;
    try {
      content = fs.readFileSync(p, 'utf8');
    } catch {
      continue;
    }
    const lines = content.split('\n');
    for (let i = 0; i < lines.length; i++) {
      if (pattern.test(lines[i]!)) {
        out.push({ file: p, line: i + 1, snippet: lines[i]!.trim(), reason });
      }
    }
  }
  return out;
}

/** Expand a list of globs into concrete file paths. Supports `**` and `*`. */
export function expandGlobs(roots: string[], extensions = ['.ts', '.tsx', '.js']): string[] {
  const out: string[] = [];
  for (const root of roots) {
    walk(root, extensions, out);
  }
  return out;
}

function walk(p: string, extensions: string[], out: string[]): void {
  let stat: fs.Stats;
  try {
    stat = fs.statSync(p);
  } catch {
    return;
  }
  if (stat.isFile()) {
    if (extensions.some((e) => p.endsWith(e))) out.push(p);
    return;
  }
  if (!stat.isDirectory()) return;
  if (path.basename(p) === 'node_modules' || path.basename(p).startsWith('.')) return;
  let entries: fs.Dirent[];
  try {
    entries = fs.readdirSync(p, { withFileTypes: true });
  } catch {
    return;
  }
  for (const e of entries) {
    walk(path.join(p, e.name), extensions, out);
  }
}

// ── Quirk-specific detection rules ────────────────────────────────────────

const NON_ASCII_STRING_LITERAL = /["'][^"']*[\x80-\xff][^"']*["']/;
const EIP712_KEYWORD = /eip712|signtyped|signEthTyped/i;

/**
 * E1: Non-ASCII string literals in files that touch EIP-712 / signTyped APIs.
 * Two-pass: first ensure the file mentions an EIP-712 keyword, then scan for
 * non-ASCII string literals inside it. Keeps noise low — unrelated umlauts
 * (i18n strings) in other files are ignored.
 */
export function detectNonAsciiInEIP712Literals(paths: string[]): DetectFinding[] {
  const out: DetectFinding[] = [];
  for (const p of paths) {
    let content: string;
    try {
      content = fs.readFileSync(p, 'utf8');
    } catch {
      continue;
    }
    if (!EIP712_KEYWORD.test(content)) continue;
    const lines = content.split('\n');
    for (let i = 0; i < lines.length; i++) {
      if (NON_ASCII_STRING_LITERAL.test(lines[i]!)) {
        out.push({
          file: p,
          line: i + 1,
          snippet: lines[i]!.trim(),
          reason:
            'BitBox firmware rejects non-ASCII in EIP-712 string values; transliterate to ASCII before signing',
        });
      }
    }
  }
  return out;
}

/** P2: BLE-dedup ordering — contains() must run BEFORE removeAll(). */
export function detectBitBoxDedupOrder(paths: string[]): DetectFinding[] {
  const out: DetectFinding[] = [];
  for (const p of paths) {
    let content: string;
    try {
      content = fs.readFileSync(p, 'utf8');
    } catch {
      continue;
    }
    const containsIdx = content.search(/seenPackets\.(has|contains|includes)\s*\(/);
    const removeIdx = content.search(/seenPackets\.(clear|removeAll|delete)\s*\(/);
    if (containsIdx !== -1 && removeIdx !== -1 && removeIdx < containsIdx) {
      const line = 1 + content.substring(0, removeIdx).split('\n').length - 1;
      out.push({
        file: p,
        line,
        snippet: content.split('\n')[line - 1]?.trim() ?? '',
        reason: 'contains() must be evaluated before removeAll() in BLE packet de-dup; reversing the order silently drops legitimate retransmits',
      });
    }
  }
  return out;
}

const HARDCODED_10S_TIMEOUT = /setTimeout\s*\([^,)]+,\s*10[\s_]*0+\s*\)|10\s*\*\s*1000|10[._]?seconds|10000\s*\)/;

/** A2: Hard-coded 10-second timeouts in transport code. */
export function detectNoHardcoded10sTimeout(paths: string[]): DetectFinding[] {
  return scan(
    paths,
    HARDCODED_10S_TIMEOUT,
    'hard-coded 10s timeouts in transport code block long user-confirm flows; use context-driven deadlines',
  );
}
