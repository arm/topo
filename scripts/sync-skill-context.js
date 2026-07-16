const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const skills_dir = path.join(root, "skills");

const check = process.argv.includes("--check");

const shared_context_syncs = {
  source: path.join(skills_dir, "_shared", "topo-project-context.md"),
  destinations: fs
    .readdirSync(skills_dir, { withFileTypes: true })
    .filter(
      (entry) => entry.isDirectory() && entry.name.startsWith("topo-project-"),
    )
    .map((entry) => path.join(skills_dir, entry.name))
    .filter((skill_dir) => fs.existsSync(path.join(skill_dir, "SKILL.md")))
    .map((skill_dir) =>
      path.join(skill_dir, "references", "topo-project-context.md"),
    ),
};

const specific_syncs = [
  {
    source: path.join(
      root,
      "docs",
      "project-specification",
      "04-build-optimization.md",
    ),
    destinations: [
      path.join(
        skills_dir,
        "topo-project-optimize-deployment",
        "references",
        "docker-build-performance.md",
      ),
    ],
  },
];

const all_syncs = [shared_context_syncs, ...specific_syncs];

let stale = false;

for (const { source, destinations } of all_syncs) {
  const canonical = fs.readFileSync(source, "utf8");

  for (const dest of destinations) {
    const relative_path = path.relative(root, dest);
    const current = fs.existsSync(dest) ? fs.readFileSync(dest, "utf8") : null;

    if (current !== canonical) {
      stale = true;

      if (check) {
        const state = current === null ? "missing" : "stale";
        console.error(`${relative_path} is ${state}`);
        continue;
      }

      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.writeFileSync(dest, canonical);
      console.log(`Updated ${relative_path}`);
    }
  }
}

if (check && stale) {
  console.error(
    "Run `npm run sync:skills` to update skill context references.",
  );
  process.exit(1);
}
