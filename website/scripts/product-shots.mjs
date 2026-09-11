// Regenerate website screenshots from the real app renderer and isolated fixtures.
// Requires Go, freeze v0.2.2, and the website's installed dependencies.
import { spawn } from "node:child_process";
import { mkdtemp, mkdir, readFile, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const site = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const root = path.resolve(site, "..");
const scratch = await mkdtemp(path.join(tmpdir(), "floor-website-shots-"));
const sizes = {};
const env = { ...process.env, XDG_CONFIG_HOME: path.join(scratch, "config"), THEFLOOR_HOME: path.join(scratch, "home") };
const run = (command, args, options = {}) => new Promise((resolve, reject) => {
  // A pipe (even an empty one) makes freeze ignore its input filename.
  const child = spawn(command, args, { cwd: root, env, ...options, stdio: ["ignore", "pipe", "pipe"] });
  let stdout = "", stderr = "";
  child.stdout.on("data", (chunk) => { stdout += chunk; });
  child.stderr.on("data", (chunk) => { stderr += chunk; });
  child.on("error", reject);
  child.on("close", (code) => {
    if (code === 0) resolve({ stdout, stderr });
    else reject(Object.assign(new Error(`${command} exited ${code}: ${stderr || stdout}`), { stdout, stderr }));
  });
});
const font = path.join(root, "mobile/assets/fonts/JetBrainsMono-Regular.ttf");
const docsOnly = process.argv.includes("--docs");
const galleryOnly = process.argv.includes("--gallery");
const dark = "#09121f";

async function batch(items, work) {
  const errors = [];
  for (let i = 0; i < items.length; i += 4) {
    const results = await Promise.allSettled(items.slice(i, i + 4).map(work));
    const failed = results.filter((r) => r.status === "rejected");
    errors.push(...failed.map((r) => r.reason));
  }
  if (errors.length) throw new AggregateError(errors);
}

async function render(ansi, relative, background = dark, thumbnail = false) {
  const key = relative.replaceAll("/", "-");
  const input = path.join(scratch, `${key}.ansi`);
  const svg = path.join(scratch, `${key}.svg`);
  await writeFile(input, ansi.trimEnd());
  await run("freeze", [input, "-c", "base", "--background", background,
    "--font.size", "16", "--font.family", "JetBrains Mono", "--font.file", font,
    "--line-height", "1.45", "--padding", "16", "--margin", "0", "--window=false", "-o", svg]);
  const destination = path.join(site, "public/shots", relative);
  await mkdir(path.dirname(destination), { recursive: true });
  const raster = sharp(svg).resize({ width: 1920, ...(relative.startsWith("themes/") ? { height: 1329, fit: "contain", background } : {}) }).flatten({ background });
  const info = relative.endsWith(".png")
    ? await raster.png({ palette: true }).toFile(destination)
    : await raster.webp({ lossless: true }).toFile(destination);
  sizes[`/shots/${relative}`] = { width: info.width, height: info.height };
  if (thumbnail) {
    await sharp(destination).resize({ width: 384 }).webp({ quality: 85 }).toFile(destination.replace(".webp", "-thumb.webp"));
  }
  console.log(`Rendered ${relative} (${info.width} × ${info.height})`);
}

function frame(output, number, marker, name) {
  const frames = [];
  let current;
  for (const line of output.split("\n")) {
    if (line.startsWith("===== UI SHOT")) {
      if (current) { frames.push(current.join("\n")); current = undefined; }
      else current = [];
    } else if (current) current.push(line);
  }
  const shot = frames[number - 1];
  const plain = shot?.replace(/\x1b\[[0-9;:]*[a-zA-Z]/g, "");
  if (!shot || shot.split("\n").length !== 32 || !plain.includes(marker)) {
    throw new Error(`${name}: frame ${number} missing expected 32-row content (${marker}). Capture: ${path.join(scratch, `${name}.txt`)}`);
  }
  return shot;
}

// Named scenarios match the documentation, using the same fixtures as uishot.
const docs = [
  ["office-overview", ["--tab", "agents"], 1, "AGENTS"],
  ["first-run-chat", ["--tab", "chat"], 1, "boss is typing"],
  ["chat-thinking", ["--think", "--think-stop", "mid"], 1, "thinking · 2 lines"],
  ["work-threads", ["--threads"], 1, "✓ done"],
  ["permission-modal", ["--tab", "chat", "--at", "2920"], 1, "PERMISSION"],
  ["question-modal", ["--ask-answer"], 1, "boss asks"],
  ["concierge", ["--concierge"], 1, "concierge"],
  ["batch-dispatch", ["--batch"], 1, "backlog dispatched"],
  ["terminal-tab", ["--terminal"], 2, "keys received: 4"],
  ["git-tab", ["--tab", "git"], 1, "untr"],
  ["theme-dracula", ["--theme", "dracula", "--tab", "agents"], 1, "AGENTS"],
  ["model-picker", ["--modelshot"], 1, "BOSS MODEL"],
];

try {
  console.log("Building isolated screenshot tools…");
  await run("go", ["build", "-o", path.join(scratch, "uishot"), "./cmd/uishot"]);
  const ui = (args, options) => run(path.join(scratch, "uishot"), args, options);
  if (!docsOnly) {
    const project = path.join(scratch, "theboringfloor");
    await mkdir(project);
    const palettes = JSON.parse((await ui(["--theme-catalog"])).stdout);
    await writeFile(path.join(site, "lib/theme-palettes.json"), JSON.stringify(palettes, null, 2) + "\n");
    await batch(palettes, async (theme) => {
      const { stdout } = await ui(["--cockpit", "--theme", theme.id], { cwd: project });
      await render(stdout, `themes/${theme.id}.webp`, theme.background, true);
    });
    await run("go", ["build", "-o", path.join(scratch, "workspaceshot"), "./cmd/workspaceshot"]);
    const workspaces = path.join(scratch, "workspaces");
    await run(path.join(scratch, "workspaceshot"), ["--out", workspaces, "--theme", "github-light", "--width", "168", "--height", "46"]);
    await batch(["floors", "board", "ticket", "files", "transcript", "expanded-transcript", "plan", "new-conversation"], async (name) => {
      await render(await readFile(path.join(workspaces, `${name}.ansi`), "utf8"), `workspaces/${name}.webp`, "#ffffff");
    });
  }
  if (!galleryOnly) {
    // The Claude integration proof exercises CLI protocol timing. Website
    // captures use a render-only conversation fixture instead.
    await run("go", ["build", "-o", path.join(scratch, "workspaceshot"), "./cmd/workspaceshot"]);
    const claude = path.join(scratch, "claude");
    await run(path.join(scratch, "workspaceshot"), ["--out", claude, "--theme", "cockpit", "--backend", "claudecode", "--width", "168", "--height", "46"]);
    await render(await readFile(path.join(claude, "plan.ansi"), "utf8"), "docs/backend-claude.png");
    await render(await readFile(path.join(claude, "board.ansi"), "utf8"), "docs/board-sync.png");
    const currentDocs = path.join(scratch, "current-docs");
    await ui(["--website-out", currentDocs]);
    await batch(["layout-normal", "layout-compact", "layout-wide", "plan-gated", "plan-presented", "thread-focus", "slash-popover", "stop-unwind"], async (name) => {
      await render(await readFile(path.join(currentDocs, `${name}.ansi`), "utf8"), `docs/${name}.png`);
    });
    await batch(docs, async ([name, flags, number, marker]) => {
      const shotEnv = { ...env, THEFLOOR_HOME: path.join(scratch, `home-${name}`) };
      const output = (await ui(flags, { env: shotEnv })).stdout;
      await writeFile(path.join(scratch, `${name}.txt`), output);
      await render(frame(output, number, marker, name), `docs/${name}.png`, name === "theme-dracula" ? "#1e1f29" : dark);
    });
  }
  let previous = {};
  try { previous = JSON.parse(await readFile(path.join(site, "lib/shot-sizes.json"), "utf8")); } catch {}
  await writeFile(path.join(site, "lib/shot-sizes.json"), JSON.stringify({ ...previous, ...sizes }, null, 2) + "\n");
  await rm(scratch, { recursive: true });
  console.log("Screenshot assets and dimensions updated.");
} catch (error) {
  console.error(error);
  console.error(`Capture files retained at ${scratch}`);
  process.exitCode = 1;
}
