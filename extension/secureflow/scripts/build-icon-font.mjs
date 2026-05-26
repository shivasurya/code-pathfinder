import svgtofont from 'svgtofont';
import svgpath from 'svgpath';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import fs from 'node:fs/promises';
import os from 'node:os';

const here = path.dirname(fileURLToPath(import.meta.url));
const sourceSvg = path.join(here, 'icons', 'codepathfinder-logo.svg');
const dist = path.join(here, '..', 'resources', 'icon-font');

// svgicons2svgfont (used by svgtofont) ignores <path transform="…"> attributes,
// drops all but the first path, and assumes viewBox starts at (0,0). The source
// SVG (pathfinder-activitybar.svg) violates all three. Flatten it first:
//   - apply each path's transform into its `d` data via svgpath
//   - translate everything by -viewBoxOrigin
//   - merge all paths into a single `d`
const raw = await fs.readFile(sourceSvg, 'utf-8');

const viewBoxMatch = raw.match(/viewBox="([\d.\-\s]+)"/);
if (!viewBoxMatch) throw new Error('Source SVG missing viewBox');
const [vbX, vbY, vbW, vbH] = viewBoxMatch[1].trim().split(/\s+/).map(Number);

const pathRe = /<path\s+([^>]*)\/>/g;
const attrRe = (name) => new RegExp(`${name}="([^"]*)"`);
const merged = [];
for (const m of raw.matchAll(pathRe)) {
  const attrs = m[1];
  const d = attrs.match(attrRe('d'))?.[1];
  if (!d) continue;
  const transform = attrs.match(attrRe('transform'))?.[1] ?? '';
  let p = svgpath(d);
  if (transform) p = p.transform(transform);
  p = p.translate(-vbX, -vbY);
  merged.push(p.toString());
}
if (!merged.length) throw new Error('No <path> elements found in source SVG');

const flatSvg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${vbW} ${vbH}">
  <path d="${merged.join(' ')}" fill="currentColor"/>
</svg>`;

const tmpSrc = await fs.mkdtemp(
  path.join(os.tmpdir(), 'codepathfinder-icons-')
);
await fs.writeFile(path.join(tmpSrc, 'codepathfinder-logo.svg'), flatSvg);

await svgtofont({
  src: tmpSrc,
  dist,
  fontName: 'codepathfinder-icons',
  startUnicode: 0xe001,
  emptyDist: true,
  generateInfoData: false,
  website: null,
  outSVGReact: false,
  outSVGReactNative: false,
  outSVGPath: false,
  css: false,
  typescript: false,
  excludeFormat: ['eot', 'svg', 'ttf']
});

await fs.rm(tmpSrc, { recursive: true, force: true });

const keep = new Set([
  'codepathfinder-icons.woff',
  'codepathfinder-icons.woff2'
]);
for (const entry of await fs.readdir(dist)) {
  if (!keep.has(entry)) {
    await fs.rm(path.join(dist, entry), { recursive: true, force: true });
  }
}

console.log('Icon font built at:', dist);
