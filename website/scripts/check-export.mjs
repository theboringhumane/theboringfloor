import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

const root = path.resolve("out");
async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nested = await Promise.all(
    entries.map((entry) =>
      entry.isDirectory()
        ? walk(path.join(directory, entry.name))
        : path.join(directory, entry.name),
    ),
  );
  return nested.flat();
}
const files = await walk(root);
const fileSet = new Set(files);
const pages = files.filter((file) => file.endsWith(".html"));
const failures = [];
let checked = 0;
const redirects = new Set(["/install.sh", "/install.ps1"]);
for (const page of pages) {
  const html = await readFile(page, "utf8");
  const relative = path.relative(root, page).replaceAll(path.sep, "/");
  const headingCount = [...html.matchAll(/<h1\b/g)].length;
  if (headingCount !== 1)
    failures.push(`${relative}: expected one h1, found ${headingCount}`);
  const pageUrl = `https://local.test/${relative.replace(/index\.html$/, "")}`;
  for (const tag of html.matchAll(/<(?:a|img|script|link|source)\b[^>]*>/g)) {
    const attribute = tag[0].match(/\b(?:href|src)="([^"]+)"/);
    if (!attribute) continue;
    const url = new URL(attribute[1].replaceAll("&amp;", "&"), pageUrl);
    if (url.origin !== "https://local.test" || redirects.has(url.pathname))
      continue;
    // The shared skip link focuses the page's <main> through its click handler.
    if (url.hash === "#main-content") continue;
    checked++;
    const filename = path.join(root, decodeURIComponent(url.pathname));
    const target = [
      filename,
      path.join(filename, "index.html"),
      `${filename}.html`,
    ].find((candidate) => fileSet.has(candidate));
    if (!target) failures.push(`${relative}: missing ${url.pathname}`);
    else if (url.hash && target.endsWith(".html")) {
      const destination = await readFile(target, "utf8");
      const id = decodeURIComponent(url.hash.slice(1));
      if (!destination.includes(`id="${id}"`))
        failures.push(`${relative}: missing anchor ${url.pathname}${url.hash}`);
    }
  }
}
if (failures.length) {
  console.error([...new Set(failures)].join("\n"));
  process.exitCode = 1;
} else
  console.log(
    `Checked ${checked} local links and assets across ${pages.length} exported pages. No broken references.`,
  );
