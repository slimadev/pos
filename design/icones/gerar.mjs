// Gera todos os ícones das apps a partir de admin.svg, pdv.svg e android.svg:
//   - cmd/admin/build/appicon.png + windows/icon.ico   (exe da administração)
//   - cmd/app/build/appicon.png   + windows/icon.ico   (exe do PDV)
//   - cmd/android/app/src/main/res/mipmap-*/ic_launcher(_round).png
//
//   cd design/icones && npm install && node gerar.mjs
import sharp from "sharp";
import pngToIco from "png-to-ico";
import { mkdir, writeFile, readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const aqui = path.dirname(fileURLToPath(import.meta.url));
const cmd = path.resolve(aqui, "../../gestao-retalho/cmd");

async function png(svg, tamanho) {
  return sharp(svg, { density: 384 }).resize(tamanho, tamanho).png().toBuffer();
}

async function exe(svgNome, pasta) {
  const svg = await readFile(path.join(aqui, svgNome));
  const build = path.join(cmd, pasta, "build");
  await writeFile(path.join(build, "appicon.png"), await png(svg, 1024));
  const tamanhos = [16, 24, 32, 48, 64, 128, 256];
  const ico = await pngToIco(await Promise.all(tamanhos.map((t) => png(svg, t))));
  await writeFile(path.join(build, "windows", "icon.ico"), ico);
  console.log(`${pasta}: appicon.png + icon.ico`);
}

// Android 7.1 (API 25) não tem ícones adaptativos: PNG por densidade.
// O launcher do Sunmi usa o ícone redondo; os do sistema (Play Store, Maps,
// Gmail) são discos claros a encher a área, por isso android.svg segue o
// mesmo desenho — o disco escuro do .exe sumia no fundo escuro do terminal.
async function android(svgNome) {
  const svg = await readFile(path.join(aqui, svgNome));
  const res = path.join(cmd, "android/app/src/main/res");
  const densidades = { mdpi: 48, hdpi: 72, xhdpi: 96, xxhdpi: 144, xxxhdpi: 192 };
  for (const [d, t] of Object.entries(densidades)) {
    const dir = path.join(res, `mipmap-${d}`);
    await mkdir(dir, { recursive: true });
    const img = await png(svg, t);
    await writeFile(path.join(dir, "ic_launcher.png"), img);
    await writeFile(path.join(dir, "ic_launcher_round.png"), img);
  }
  console.log("android: mipmap-* ic_launcher(_round).png");
}

await exe("admin.svg", "admin");
await exe("pdv.svg", "app");
await android("android.svg");
