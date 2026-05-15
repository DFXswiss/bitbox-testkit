import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';
import {
  detectNonAsciiInEIP712Literals,
  detectBitBoxDedupOrder,
  detectNoHardcoded10sTimeout,
  expandGlobs,
} from '../src/guards/index.js';

function inTempDir(files: Record<string, string>, fn: (root: string) => void): void {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'bbtk-test-'));
  for (const [rel, content] of Object.entries(files)) {
    const full = path.join(root, rel);
    fs.mkdirSync(path.dirname(full), { recursive: true });
    fs.writeFileSync(full, content);
  }
  try {
    fn(root);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

describe('detectNonAsciiInEIP712Literals', () => {
  it('flags non-ASCII in EIP-712 context', () => {
    inTempDir(
      {
        'bad.ts': 'const msg = "EIP712 hëllo"; signEthTypedData(msg);',
      },
      (root) => {
        const paths = expandGlobs([root]);
        const findings = detectNonAsciiInEIP712Literals(paths);
        expect(findings.length).toBeGreaterThan(0);
      },
    );
  });

  it('does not flag clean ASCII', () => {
    inTempDir(
      {
        'ok.ts': 'const msg = "EIP712 hello"; signEthTypedData(msg);',
      },
      (root) => {
        const paths = expandGlobs([root]);
        expect(detectNonAsciiInEIP712Literals(paths)).toEqual([]);
      },
    );
  });

  it('ignores non-ASCII unrelated to signing', () => {
    inTempDir(
      {
        'unrelated.ts': 'const greeting = "Hëllo from i18n";',
      },
      (root) => {
        const paths = expandGlobs([root]);
        expect(detectNonAsciiInEIP712Literals(paths)).toEqual([]);
      },
    );
  });
});

describe('detectBitBoxDedupOrder', () => {
  it('flags removeAll-before-contains pattern', () => {
    inTempDir(
      {
        'buggy.ts': `
function dedup(id: string) {
  seenPackets.clear();
  if (seenPackets.has(id)) return;
}`,
      },
      (root) => {
        const paths = expandGlobs([root]);
        const findings = detectBitBoxDedupOrder(paths);
        expect(findings.length).toBeGreaterThan(0);
      },
    );
  });

  it('passes correct order', () => {
    inTempDir(
      {
        'fixed.ts': `
function dedup(id: string) {
  if (seenPackets.has(id)) return;
  seenPackets.clear();
}`,
      },
      (root) => {
        const paths = expandGlobs([root]);
        expect(detectBitBoxDedupOrder(paths)).toEqual([]);
      },
    );
  });
});

describe('detectNoHardcoded10sTimeout', () => {
  it('flags setTimeout(..., 10000)', () => {
    inTempDir(
      {
        'bad.ts': `
function poll() {
  setTimeout(cb, 10000);
}`,
      },
      (root) => {
        const paths = expandGlobs([root]);
        const findings = detectNoHardcoded10sTimeout(paths);
        expect(findings.length).toBeGreaterThan(0);
      },
    );
  });

  it('flags 10 * 1000 multiplications', () => {
    inTempDir(
      {
        'bad.ts': `setTimeout(cb, 10 * 1000);`,
      },
      (root) => {
        const paths = expandGlobs([root]);
        expect(detectNoHardcoded10sTimeout(paths).length).toBeGreaterThan(0);
      },
    );
  });
});

describe('expandGlobs', () => {
  it('walks directories and filters extensions', () => {
    inTempDir(
      {
        'src/a.ts': '',
        'src/b.js': '',
        'src/c.json': '',
        'node_modules/x.ts': '',
      },
      (root) => {
        const paths = expandGlobs([root]);
        expect(paths.length).toBe(2); // a.ts + b.js; .json excluded, node_modules skipped
        expect(paths.every((p) => !p.includes('node_modules'))).toBe(true);
      },
    );
  });
});
