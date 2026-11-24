/**
 * Engine visual de imagens e metadados visuais de produtos estilo Zobaze POS para o Admin.
 */

export interface VisualProduto {
  imagemUrl: string;
  gradiente: string;
  corAcento: string;
  categoriaNome: string;
  icone: string;
  descricaoCurta: string;
}

const IMAGENS_REAIS: Record<string, string> = {
  // Bebidas e Cervejas
  cerveja: "https://images.unsplash.com/photo-1608270199042-4f0163359d99?w=400&q=80&auto=format&fit=crop",
  "2m": "https://images.unsplash.com/photo-1535958636474-b021ee887b13?w=400&q=80&auto=format&fit=crop",
  laurentina: "https://images.unsplash.com/photo-1518176258769-f227c798150e?w=400&q=80&auto=format&fit=crop",
  "castle lite": "https://images.unsplash.com/photo-1608270199042-4f0163359d99?w=400&q=80&auto=format&fit=crop",
  heineken: "https://images.unsplash.com/photo-1618885472179-5e474019f2df?w=400&q=80&auto=format&fit=crop",
  whisky: "https://images.unsplash.com/photo-1527281400683-1aae777175f8?w=400&q=80&auto=format&fit=crop",
  jameson: "https://images.unsplash.com/photo-1527281400683-1aae777175f8?w=400&q=80&auto=format&fit=crop",
  gin: "https://images.unsplash.com/photo-1551538827-9c037cb4f32a?w=400&q=80&auto=format&fit=crop",
  bombay: "https://images.unsplash.com/photo-1551538827-9c037cb4f32a?w=400&q=80&auto=format&fit=crop",
  vinho: "https://images.unsplash.com/photo-1510812431401-41d2bd2722f3?w=400&q=80&auto=format&fit=crop",
  refrigerante: "https://images.unsplash.com/photo-1622483767028-3f66f32aef97?w=400&q=80&auto=format&fit=crop",
  "coca-cola": "https://images.unsplash.com/photo-1622483767028-3f66f32aef97?w=400&q=80&auto=format&fit=crop",
  fanta: "https://images.unsplash.com/photo-1624517452488-04869289c4ca?w=400&q=80&auto=format&fit=crop",
  sprite: "https://images.unsplash.com/photo-1625772299848-391b6a87d7b3?w=400&q=80&auto=format&fit=crop",
  agua: "https://images.unsplash.com/photo-1548839140-29a749e1bc4e?w=400&q=80&auto=format&fit=crop",
  água: "https://images.unsplash.com/photo-1548839140-29a749e1bc4e?w=400&q=80&auto=format&fit=crop",
  sumo: "https://images.unsplash.com/photo-1613478223719-2ab802602423?w=400&q=80&auto=format&fit=crop",

  // Cozinha e Petiscos
  frango: "https://images.unsplash.com/photo-1598515214211-89d3c73ae83b?w=400&q=80&auto=format&fit=crop",
  batata_frita: "https://images.unsplash.com/photo-1576107232684-1279f3908594?w=400&q=80&auto=format&fit=crop",
  asinha: "https://images.unsplash.com/photo-1567620832903-9fc6debc209f?w=400&q=80&auto=format&fit=crop",
  petisco: "https://images.unsplash.com/photo-1541544741938-0af808871cc0?w=400&q=80&auto=format&fit=crop",

  // Mercearia e Alimentos
  arroz: "https://images.unsplash.com/photo-1586201375761-83865001e31c?w=400&q=80&auto=format&fit=crop",
  oleo: "https://images.unsplash.com/photo-1474979266404-7eaacbcd87c5?w=400&q=80&auto=format&fit=crop",
  óleo: "https://images.unsplash.com/photo-1474979266404-7eaacbcd87c5?w=400&q=80&auto=format&fit=crop",
  acucar: "https://images.unsplash.com/photo-1581441363689-1f3c3c414635?w=400&q=80&auto=format&fit=crop",
  açúcar: "https://images.unsplash.com/photo-1581441363689-1f3c3c414635?w=400&q=80&auto=format&fit=crop",
  farinha: "https://images.unsplash.com/photo-1509440159596-0249088772ff?w=400&q=80&auto=format&fit=crop",
  milho: "https://images.unsplash.com/photo-1551754655-cd27e38d2076?w=400&q=80&auto=format&fit=crop",
  feijao: "https://images.unsplash.com/photo-1551462147-37885acc36f1?w=400&q=80&auto=format&fit=crop",
  feijão: "https://images.unsplash.com/photo-1551462147-37885acc36f1?w=400&q=80&auto=format&fit=crop",
  batata: "https://images.unsplash.com/photo-1518977676601-b53f82aba655?w=400&q=80&auto=format&fit=crop",
  cebola: "https://images.unsplash.com/photo-1518977822534-7049a61ee0c2?w=400&q=80&auto=format&fit=crop",
  tomate: "https://images.unsplash.com/photo-1592924357228-91a4daadcfea?w=400&q=80&auto=format&fit=crop",
  leite: "https://images.unsplash.com/photo-1550583724-b2692b85b150?w=400&q=80&auto=format&fit=crop",
  bolacha: "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?w=400&q=80&auto=format&fit=crop",
  biscoito: "https://images.unsplash.com/photo-1499636136210-6f4ee915583e?w=400&q=80&auto=format&fit=crop",
  sabao: "https://images.unsplash.com/photo-1607006314144-8cb3b80b72fa?w=400&q=80&auto=format&fit=crop",
  sabão: "https://images.unsplash.com/photo-1607006314144-8cb3b80b72fa?w=400&q=80&auto=format&fit=crop",

  // Farmácia e Higiene
  paracetamol: "https://images.unsplash.com/photo-1584308666744-24d5c474f2ae?w=400&q=80&auto=format&fit=crop",
  medicamento: "https://images.unsplash.com/photo-1471864190281-a93a3070b6de?w=400&q=80&auto=format&fit=crop",
  alcool: "https://images.unsplash.com/photo-1584515979956-d9f6e5d09982?w=400&q=80&auto=format&fit=crop",
  álcool: "https://images.unsplash.com/photo-1584515979956-d9f6e5d09982?w=400&q=80&auto=format&fit=crop",
  ibuprofeno: "https://images.unsplash.com/photo-1584017911766-d451b3d0e843?w=400&q=80&auto=format&fit=crop",
  amoxicilina: "https://images.unsplash.com/photo-1584308666744-24d5c474f2ae?w=400&q=80&auto=format&fit=crop",
  vitamina: "https://images.unsplash.com/photo-1550572017-edd951aa8f72?w=400&q=80&auto=format&fit=crop",
  xarope: "https://images.unsplash.com/photo-1631549916768-4119b2e5f926?w=400&q=80&auto=format&fit=crop",
  penso: "https://images.unsplash.com/photo-1583947215259-38e31be8751f?w=400&q=80&auto=format&fit=crop",
  higiene: "https://images.unsplash.com/photo-1584515979956-d9f6e5d09982?w=400&q=80&auto=format&fit=crop",

  // Ferragem e Construção
  cimento: "https://images.unsplash.com/photo-1589939705384-5185137a7f0f?w=400&q=80&auto=format&fit=crop",
  areia: "https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=400&q=80&auto=format&fit=crop",
  cabo: "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=400&q=80&auto=format&fit=crop",
  eletrico: "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=400&q=80&auto=format&fit=crop",
  elétrico: "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=400&q=80&auto=format&fit=crop",
  lampada: "https://images.unsplash.com/photo-1550684848-fac1c5b4e853?w=400&q=80&auto=format&fit=crop",
  lâmpada: "https://images.unsplash.com/photo-1550684848-fac1c5b4e853?w=400&q=80&auto=format&fit=crop",
  chave: "https://images.unsplash.com/photo-1581783342308-f792dbdd27c5?w=400&q=80&auto=format&fit=crop",
  ferramenta: "https://images.unsplash.com/photo-1581783342308-f792dbdd27c5?w=400&q=80&auto=format&fit=crop",
  martelo: "https://images.unsplash.com/photo-1586864387967-d02ef85d93e8?w=400&q=80&auto=format&fit=crop",
  tinta: "https://images.unsplash.com/photo-1589939705384-5185137a7f0f?w=400&q=80&auto=format&fit=crop",
  prego: "https://images.unsplash.com/photo-1584308666744-24d5c474f2ae?w=400&q=80&auto=format&fit=crop",
};

export function gerarSvgFallback(nome: string, cor: string, textoCurto: string): string {
  const inicial = (nome || "P").trim().charAt(0).toUpperCase();
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" width="100%" height="100%">
    <defs>
      <linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stop-color="${cor}" stop-opacity="0.9" />
        <stop offset="100%" stop-color="#111827" stop-opacity="0.95" />
      </linearGradient>
    </defs>
    <rect width="200" height="200" fill="url(#g)" rx="16" />
    <circle cx="100" cy="90" r="46" fill="rgba(255,255,255,0.12)" />
    <text x="100" y="106" font-family="system-ui, -apple-system, sans-serif" font-size="44" font-weight="800" fill="#ffffff" text-anchor="middle">${inicial}</text>
    <text x="100" y="155" font-family="system-ui, -apple-system, sans-serif" font-size="12" font-weight="600" fill="rgba(255,255,255,0.85)" text-anchor="middle">${textoCurto.slice(0, 16)}</text>
  </svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

export function obterVisualProduto(
  nome: string,
  categoria?: string | null,
  tipoNegocio?: string
): VisualProduto {
  const normal = (nome || "").toLowerCase().trim();
  const catNormal = (categoria || "").toLowerCase().trim();

  let imagemUrl = "";
  for (const [chave, url] of Object.entries(IMAGENS_REAIS)) {
    if (normal.includes(chave) || catNormal.includes(chave)) {
      imagemUrl = url;
      break;
    }
  }

  let corAcento = "#3b82f6";
  let gradiente = "linear-gradient(135deg, #1e3a8a 0%, #3b82f6 100%)";
  let icone = "ShoppingBag";

  if (
    catNormal.includes("bebida") ||
    catNormal.includes("cerveja") ||
    tipoNegocio === "bottlestore" ||
    normal.includes("2m") ||
    normal.includes("whisky")
  ) {
    corAcento = "#eab308";
    gradiente = "linear-gradient(135deg, #713f12 0%, #ca8a04 100%)";
    icone = "Sparkles";
    if (!imagemUrl) {
      imagemUrl = "https://images.unsplash.com/photo-1510812431401-41d2bd2722f3?w=400&q=80&auto=format&fit=crop";
    }
  } else if (
    catNormal.includes("medicamento") ||
    catNormal.includes("farmacia") ||
    catNormal.includes("higiene") ||
    tipoNegocio === "farmacia"
  ) {
    corAcento = "#06b6d4";
    gradiente = "linear-gradient(135deg, #164e63 0%, #0891b2 100%)";
    icone = "Beaker";
    if (!imagemUrl) {
      imagemUrl = "https://images.unsplash.com/photo-1471864190281-a93a3070b6de?w=400&q=80&auto=format&fit=crop";
    }
  } else if (
    catNormal.includes("ferramenta") ||
    catNormal.includes("cimento") ||
    catNormal.includes("elétrico") ||
    tipoNegocio === "ferragem"
  ) {
    corAcento = "#f97316";
    gradiente = "linear-gradient(135deg, #7c2d12 0%, #ea580c 100%)";
    icone = "Wrench";
    if (!imagemUrl) {
      imagemUrl = "https://images.unsplash.com/photo-1581783342308-f792dbdd27c5?w=400&q=80&auto=format&fit=crop";
    }
  } else {
    corAcento = "#10b981";
    gradiente = "linear-gradient(135deg, #064e3b 0%, #059669 100%)";
    icone = "BuildingStorefront";
    if (!imagemUrl) {
      imagemUrl = "https://images.unsplash.com/photo-1542838132-92c53300491e?w=400&q=80&auto=format&fit=crop";
    }
  }

  return {
    imagemUrl,
    gradiente,
    corAcento,
    categoriaNome: categoria || "Geral",
    icone,
    descricaoCurta: nome,
  };
}
