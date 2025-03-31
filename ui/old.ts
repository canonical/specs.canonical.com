import { type BunPlugin } from "bun";
import * as sass from "sass";

const outDir = "dist";

const style: BunPlugin = {
  name: "Sass Loader",
  async setup(build) {
    const path = build.config.entrypoints[0];
    const fileContent = await globalThis.Bun.file(path).text();
    const contents = sass.compileString(fileContent);
    globalThis.Bun.write(`${outDir}/assets/style.css`, contents.css);

    console.log("Sass code compiled!");
  },
};
async function copy(src: string, dest: string) {
  const file = Bun.file(src);
  await Bun.write(dest, file);
}

await copy("index.html", `${outDir}/index.html`).then(() =>
  console.log("Bundling the app...")
);

await Bun.build({
  entrypoints: ["src/main.tsx"],
  outdir: `${outDir}/assets`,
  minify: true,
})
  .then(
    async () =>
      await Bun.build({
        entrypoints: ["src/app.scss"],
        outdir: `${outDir}/assets`,
        minify: true,
        plugins: [style],
      })
  )
  .then(() => console.log("⚡ Build complete! ⚡"))
  .catch(() => process.exit(1));
