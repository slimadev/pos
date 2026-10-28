// Ponte de DEMONSTRAÇÃO: cria window.go.admin.AdminApp e window.go.api.App
// com a mesma forma que o Wails cria, mas devolve dados de exemplo em vez de
// chamar o Go. Serve para mostrar e validar a app no terminal (WebView
// Chrome 62) antes de a ponte real (gobridge) existir. Escrito em ES5.
//
// Há uma loja de demonstração por tipo de negócio (ferragem, mercearia,
// farmácia, bottlestore); escolhe-se no ecrã de login. Vendas e contas
// feitas no PDV ficam guardadas no localStorage e aparecem no admin.
(function () {
  "use strict";

  // `__DEMO_DESKTOP` (definido antes deste script) serve para pré-visualizar
  // o build do desktop com estes dados, sem os ajustes do terminal.
  window.__PLATAFORMA = window.__DEMO_DESKTOP ? "desktop" : "android";
  window.__DEMO = true;

  // --- Erros de JS no ecrã (não há DevTools no terminal) --------------------
  function mostrarErro(texto) {
    var el = document.getElementById("__erro-demo");
    if (!el) {
      el = document.createElement("pre");
      el.id = "__erro-demo";
      el.style.cssText =
        "position:fixed;left:0;right:0;bottom:0;max-height:40%;overflow:auto;margin:0;" +
        "padding:8px;background:#b91c1c;color:#fff;font:11px monospace;z-index:99999;white-space:pre-wrap";
      el.onclick = function () { el.parentNode.removeChild(el); };
      (document.body || document.documentElement).appendChild(el);
    }
    el.textContent += texto + "\n";
  }
  window.addEventListener("error", function (e) {
    mostrarErro("ERRO: " + e.message + " @" + e.filename + ":" + e.lineno);
  });
  window.addEventListener("unhandledrejection", function (e) {
    mostrarErro("PROMISE: " + (e.reason && e.reason.stack ? e.reason.stack : e.reason));
  });

  // --- Utilitários -----------------------------------------------------------
  function ler(chave, omissao) {
    try {
      var v = localStorage.getItem(chave);
      return v ? JSON.parse(v) : omissao;
    } catch (e) {
      return omissao;
    }
  }
  function gravar(chave, valor) {
    try { localStorage.setItem(chave, JSON.stringify(valor)); } catch (e) { /* sem storage */ }
  }
  function copia(x) { return x === undefined ? null : JSON.parse(JSON.stringify(x)); }
  function diaISO(diasAtras, hora) {
    var d = new Date();
    d.setDate(d.getDate() - diasAtras);
    if (hora !== undefined) d.setHours(hora, (diasAtras * 17) % 60, 0, 0);
    return d.toISOString();
  }
  function dia(diasAtras) { return diaISO(diasAtras).slice(0, 10); }

  // --- As quatro lojas -------------------------------------------------------
  // produtos: [nome, categoria, preço, unidade, stock, código de barras, custo?]
  var LOJAS = {
    ferragem: {
      nome: "Ferragem Central", modulos: ["pos_direto"],
      categorias: ["Cimento e areia", "Ferramentas", "Elétrico", "Canalização", "Tintas"],
      produtos: [
        ["Cimento 50kg", 1, 650, "saco", 42, "6001234500011"],
        ["Areia fina (m³)", 1, 1800, "m³", 8, null],
        ["Tijolo maciço", 1, 12, "unidade", 1500, null],
        ["Bloco de cimento 15cm", 1, 38, "unidade", 620, null],
        ["Martelo de unha 500g", 2, 450, "unidade", 12, "6001234500035"],
        ["Chave inglesa 10\"", 2, 890, "unidade", 7, "6001234500042"],
        ["Chave de fendas", 2, 120, "unidade", 30, null],
        ["Serrote 20\"", 2, 690, "unidade", 0, null],
        ["Fita métrica 5m", 2, 230, "unidade", 18, "6001234500059"],
        ["Cabo 2,5mm² (m)", 3, 95, "metro", 400, null],
        ["Disjuntor 16A", 3, 340, "unidade", 22, "6001234500066"],
        ["Lâmpada LED 9W", 3, 130, "unidade", 60, "6001234500073"],
        ["Fita isoladora", 3, 45, "unidade", 3, "6001234500080"],
        ["Tubo PVC 50mm 3m", 4, 410, "unidade", 35, null],
        ["Joelho PVC 50mm", 4, 65, "unidade", 120, null],
        ["Torneira de jardim", 4, 380, "unidade", 9, null],
        ["Tinta acrílica branca 20L", 5, 3250, "balde", 6, "6001234500097"],
        ["Pincel 2\"", 5, 85, "unidade", 40, null],
      ],
      clientes: ["Construções Tembe, Lda", "João Mabunda", "Carlos Muianga"],
    },
    mercearia: {
      nome: "Mercearia Dona Ana", modulos: ["pos_direto", "venda_granel"],
      categorias: ["Mercearia", "Bebidas", "Higiene", "Granel"],
      produtos: [
        ["Arroz 5kg", 1, 320, "unidade", 40, "6009876500011"],
        ["Óleo vegetal 1L", 1, 130, "unidade", 55, "6009876500028"],
        ["Açúcar 1kg", 1, 80, "unidade", 70, "6009876500035"],
        ["Farinha de milho 5kg", 1, 260, "unidade", 25, null],
        ["Leite UHT 1L", 1, 95, "unidade", 48, "6009876500042"],
        ["Pão (unidade)", 1, 10, "unidade", 200, null],
        ["Água 500ml", 2, 60, "unidade", 120, "6009876500059"],
        ["Coca-Cola 500ml", 2, 120, "unidade", 90, "6009876500066"],
        ["Sumo Compal 1L", 2, 210, "unidade", 2, null],
        ["Sabão em barra", 3, 45, "unidade", 80, null],
        ["Pasta de dentes", 3, 110, "unidade", 30, null],
        ["Arroz a granel", 4, 75, "kg", 180, null],
        ["Feijão manteiga", 4, 140, "kg", 60, null],
        ["Amendoim torrado", 4, 220, "kg", 15, null],
      ],
      clientes: ["Dona Rita Chissano", "Restaurante Sabor da Terra"],
    },
    farmacia: {
      nome: "Farmácia Saúde+", modulos: ["pos_direto", "lotes_validade", "receitas_medicas"],
      categorias: ["Medicamentos", "Higiene", "Bebé", "Suplementos"],
      produtos: [
        ["Paracetamol 500mg (20 comp.)", 1, 55, "caixa", 140, "5601234500011"],
        ["Ibuprofeno 400mg (20 comp.)", 1, 95, "caixa", 60, "5601234500028"],
        ["Amoxicilina 500mg (16 cáps.)", 1, 180, "caixa", 35, "5601234500035"],
        ["Xarope para a tosse 150ml", 1, 240, "frasco", 22, null],
        ["Soro oral (sais)", 1, 35, "saqueta", 300, null],
        ["Álcool gel 250ml", 2, 110, "unidade", 45, "5601234500042"],
        ["Protetor solar FPS50", 2, 690, "unidade", 12, null],
        ["Fraldas bebé T3 (30)", 3, 520, "pacote", 18, null],
        ["Leite em pó 400g", 3, 610, "lata", 4, null],
        ["Vitamina C 1000mg", 4, 320, "tubo", 30, null],
        ["Multivitamínico (30)", 4, 450, "caixa", 15, null],
      ],
      clientes: ["Maria Langa", "Clínica Boa Saúde"],
      lotes: [
        [3, "AMX-2410", 9, 12], [9, "LP-0912", 15, 4], [5, "SRO-77", 28, 120],
        [4, "XT-5531", 41, 8], [1, "PC-1180", 55, 60],
      ],
    },
    bottlestore: {
      nome: "Bottlestore XPTO", modulos: ["contas_abertas", "dividas", "fichas_tecnicas"],
      categorias: ["Cervejas", "Refrigerantes", "Destilados", "Comida"],
      produtos: [
        ["2M 550ml", 1, 180, "unidade", 240, "6004567800011"],
        ["Laurentina Preta 550ml", 1, 195, "unidade", 160, "6004567800028"],
        ["Heineken 330ml", 1, 220, "unidade", 96, null],
        ["Manica 550ml", 1, 170, "unidade", 3, null],
        ["Coca-Cola 500ml", 2, 120, "unidade", 110, null],
        ["Água 500ml", 2, 60, "unidade", 150, null],
        ["Sprite 330ml", 2, 100, "unidade", 70, null],
        ["Whisky J.W. Red (dose)", 3, 250, "dose", 40, null],
        ["Vodka Smirnoff (dose)", 3, 180, "dose", 55, null],
        ["Caipirinha de vodka", 3, 350, "copo", 30, null],
        ["Frango ¼ c/ batata", 4, 450, "prato", 18, null],
        ["Prego no pão", 4, 280, "unidade", 25, null],
        ["Amendoim (taça)", 4, 80, "taça", 40, null],
      ],
      clientes: ["Grupo do João", "Ana Cumbe", "Rui Bila"],
      contas: [
        ["Mesa 3", 0, [[1, 4], [11, 1], [5, 2]], 0.6],
        ["Grupo do João", 1, [[2, 6], [8, 3], [13, 2], [11, 2]], 2.3],
        ["Balcão 2", 0, [[1, 2], [6, 1]], 0.4],
        ["Esplanada 1", 0, [[3, 3], [10, 2]], 1.1],
        ["Aniversário Rita", 2, [[2, 10], [11, 4], [9, 6], [7, 4]], 3.2],
      ],
    },
  };

  var TIPOS = ["ferragem", "mercearia", "farmacia", "bottlestore"];
  var ROTULO = { ferragem: "Ferragem", mercearia: "Mercearia", farmacia: "Farmácia", bottlestore: "Bar" };
  var tipo = ler("demo:tipo", "ferragem");
  if (TIPOS.indexOf(tipo) < 0) tipo = "ferragem";
  var L = LOJAS[tipo];
  var prefixo = "demo:" + tipo + ":";

  // --- Modelo derivado da loja escolhida ---------------------------------
  var categorias = L.categorias.map(function (n, i) {
    return { ID: i + 1, NegocioID: 1, Nome: n, CategoriaPaiID: null, Ativo: true };
  });
  var produtos = [], unidadesTodas = [], stock = {}, catalogo = [], porUnidade = {};
  L.produtos.forEach(function (p, i) {
    var id = i + 1;
    var granel = p[3] === "kg";
    var composto = tipo === "bottlestore" && p[1] === 4 && id !== 13;
    var produto = {
      ID: id, NegocioID: 1, CategoriaID: p[1], Nome: p[0],
      TipoProduto: composto ? "composto" : "simples", UnidadeBase: granel ? "kg" : p[3],
      StockMinimo: 5, ControlaStock: !composto, LoteControlado: tipo === "farmacia" && p[1] === 1, Ativo: true,
      CriadoEm: diaISO(90), AtualizadoEm: diaISO(5),
    };
    var unidade = {
      ID: 100 + id, ProdutoID: id, NomeUnidade: p[3], QtdEmUnidadeBase: 1,
      PrecoCusto: Math.round(p[2] * 0.72), PrecoVenda: p[2], CodigoBarras: p[5],
      IsPadraoVenda: true, IsPadraoCompra: true,
    };
    produtos.push(produto);
    unidadesTodas.push(unidade);
    stock[id] = p[4];
    porUnidade[unidade.ID] = { produto: produto, unidade: unidade };
    catalogo.push({
      Produto: produto, Unidades: [unidade], UnidadePadraoID: unidade.ID,
      Disponivel: p[4], Vendido30Dias: 200 - i * 9,
    });
  });
  var armazens = [
    { ID: 1, NegocioID: 1, Nome: "Loja", Tipo: "principal", Ativo: true, ForaDeVenda: false },
    { ID: 2, NegocioID: 1, Nome: "Armazém das traseiras", Tipo: "deposito", Ativo: true, ForaDeVenda: true },
  ];
  var clientes = L.clientes.map(function (n, i) {
    return {
      ID: i + 2, NegocioID: 1, Nome: n, Telefone: "+258 84 " + (1000000 + i * 734521), NIF: null,
      Morada: null, Email: null, LimiteCredito: i === 0 ? 20000 : null, IsAnonimo: false, Ativo: true,
    };
  });
  var operadores = [
    { ID: 1, Nome: "Ana Cumbe", Ativo: true },
    { ID: 2, Nome: "Rui Bila", Ativo: true },
  ];

  // Turno do PDV na demonstração: qualquer PIN de 4 a 6 algarismos entra.
  var operadorTurno = 0;
  try { operadorTurno = Number(sessionStorage.getItem("demo-turno")) || 0; } catch (e) {}
  function nomeOperadorTurno() {
    var op = operadores.filter(function (o) { return o.ID === operadorTurno; })[0];
    return op ? op.Nome : "Administrador";
  }

  // Vendas: histórico gerado + o que o PDV registou (localStorage).
  function vendasBase() {
    var lista = [];
    var id = 900;
    for (var d = 13; d >= 0; d--) {
      var n = 2 + ((d * 7) % 4);
      for (var k = 0; k < n; k++) {
        var p = catalogo[(d * 3 + k * 5) % catalogo.length];
        var qtd = 1 + ((d + k) % 3);
        var total = p.Unidades[0].PrecoVenda * qtd;
        var status = (d + k) % 9 === 0 ? "divida" : (d + k) % 13 === 0 ? "cancelada" : "paga";
        lista.push({
          Venda: {
            ID: id++, NegocioID: 1, ArmazemID: 1, ClienteID: status === "divida" ? clientes[0].ID : 1, ContaID: null,
            TipoVenda: tipo === "bottlestore" ? "conta_fechada" : "direta", Subtotal: total, DescontoTotal: 0,
            Total: total, Status: status, CriadoEm: diaISO(d, 9 + k * 2), UtilizadorID: 1 + (k % 2),
          },
          ClienteNome: status === "divida" ? clientes[0].Nome : "Anónimo",
          OperadorNome: operadores[k % 2].Nome,
          _itens: [[p, qtd]],
        });
      }
    }
    return lista;
  }
  var vendasHistorico = vendasBase();
  var vendasPDV = ler(prefixo + "vendas", []); // [{resumo, recibo}]
  function todasVendas() {
    return vendasHistorico.concat(vendasPDV.map(function (v) { return v.resumo; }));
  }
  var vendasHoje = function () {
    var hoje = dia(0);
    return todasVendas().filter(function (v) { return v.Venda.CriadoEm.slice(0, 10) === hoje && v.Venda.Status !== "cancelada"; });
  };

  function linhasLoja() { return ["NUIT 400123456", "Av. 24 de Julho, Maputo", "+258 84 000 0000"]; }
  function recibo(id, itens, desconto, pagamentos, clienteNome) {
    var subtotal = itens.reduce(function (a, i) { return a + i.Subtotal; }, 0);
    return {
      NegocioNome: L.nome, LinhasLoja: linhasLoja(), ClienteNUIT: "", VendaID: id, CriadoEm: new Date().toISOString(),
      ClienteNome: clienteNome || "Anónimo", Itens: itens, Subtotal: subtotal, DescontoTotal: desconto || 0,
      Total: Math.max(0, subtotal - (desconto || 0)), Pagamentos: pagamentos, Papel: "58", Logo: "",
      DocumentoNome: "", DocumentoCodigo: "", TaxaIVA: 0, ValorIVA: 0,
    };
  }
  function itemRecibo(unidadeId, quantidade) {
    var u = porUnidade[unidadeId];
    return {
      NomeProduto: u.produto.Nome, NomeUnidade: u.unidade.NomeUnidade, Quantidade: quantidade,
      PrecoUnitario: u.unidade.PrecoVenda, Desconto: 0, Subtotal: u.unidade.PrecoVenda * quantidade,
    };
  }
  function registarVendaPDV(itens, desconto, pagamentos, clienteId, tipoVenda) {
    var id = 2000 + vendasPDV.length + 1;
    var cliente = clientes.filter(function (c) { return c.ID === clienteId; })[0];
    var r = recibo(id, itens, desconto, pagamentos.map(function (p) { return { Metodo: p.Metodo, Valor: p.Valor }; }), cliente && cliente.Nome);
    var resumo = {
      Venda: {
        ID: id, NegocioID: 1, ArmazemID: 1, ClienteID: clienteId || 1, ContaID: null, TipoVenda: tipoVenda,
        Subtotal: r.Subtotal, DescontoTotal: r.DescontoTotal, Total: r.Total, Status: "paga",
        CriadoEm: new Date().toISOString(), UtilizadorID: 1,
      },
      ClienteNome: cliente ? cliente.Nome : "Anónimo", OperadorNome: nomeOperadorTurno(),
    };
    vendasPDV.push({ resumo: resumo, recibo: r });
    gravar(prefixo + "vendas", vendasPDV);
    return resumo.Venda;
  }

  // Contas abertas (bottlestore).
  var contas = ler(prefixo + "contas", null);
  if (!contas) {
    contas = [];
    var proxItem = 1;
    (L.contas || []).forEach(function (c, i) {
      var conta = {
        ID: i + 1, NegocioID: 1, ClienteID: c[1] ? clientes[c[1] - 1].ID : 1, Identificador: c[0], Status: "aberta",
        AbertaEm: new Date(Date.now() - c[3] * 3600000).toISOString(), FechadaEm: null,
        UtilizadorAberturaID: 1, UtilizadorFechoID: null, MotivoCancelamento: null,
      };
      var itens = c[2].map(function (it) {
        var prod = catalogo[it[0] - 1];
        return {
          ID: proxItem++, ContaID: conta.ID, ProdutoID: prod.Produto.ID, UnidadeProdutoID: prod.Unidades[0].ID,
          Quantidade: it[1], PrecoUnitarioAplicado: prod.Unidades[0].PrecoVenda, Desconto: 0,
          CriadoEm: conta.AbertaEm, UtilizadorID: 1,
        };
      });
      contas.push({ conta: conta, itens: itens });
    });
  }
  function gravarContas() { gravar(prefixo + "contas", contas); }
  function contaPorId(id) { return contas.filter(function (c) { return c.conta.ID === id; })[0]; }
  function proximoIdItem() {
    var max = 0;
    contas.forEach(function (c) { c.itens.forEach(function (i) { if (i.ID > max) max = i.ID; }); });
    return max + 1;
  }

  // --- Respostas ----------------------------------------------------------
  var negocio = { ID: 1, Nome: L.nome, TipoNegocio: tipo, Ativo: true };
  var loja = {
    Nome: L.nome, TipoNegocio: tipo, Tema: tipo, Logo: "", PapelRecibo: "58", NUIT: "400123456",
    Endereco: "Av. 24 de Julho, Maputo", Telefone: "+258 84 000 0000", Email: "", TaxaIVA: 16,
  };

  function dashboard() {
    var serie = [];
    for (var d = 13; d >= 0; d--) {
      var t = 0;
      todasVendas().forEach(function (v) {
        if (v.Venda.CriadoEm.slice(0, 10) === dia(d) && v.Venda.Status !== "cancelada") t += v.Venda.Total;
      });
      serie.push({ Dia: dia(d), Total: t });
    }
    var hoje = vendasHoje().reduce(function (a, v) { return a + v.Venda.Total; }, 0);
    var ontem = serie[12].Total || 1;
    var abaixo = catalogo.filter(function (c) { return c.Produto.ControlaStock && c.Disponivel < 5; });
    var emConsumo = contas.reduce(function (a, c) {
      return a + c.itens.reduce(function (b, i) { return b + i.Quantidade * i.PrecoUnitarioAplicado; }, 0);
    }, 0);
    var alertas = [];
    if (abaixo.length) {
      alertas.push({
        Severidade: "alta", Titulo: abaixo.length + " produto(s) abaixo do mínimo",
        Detalhe: abaixo.map(function (c) { return c.Produto.Nome; }).join(", "),
      });
    }
    if (tipo === "farmacia") {
      alertas.push({ Severidade: "alta", Titulo: "2 lotes a vencer em 15 dias", Detalhe: "Amoxicilina 500mg (AMX-2410), Leite em pó 400g (LP-0912)" });
    }
    if (tipo === "bottlestore") {
      alertas.push({ Severidade: "media", Titulo: "Conta aberta há mais de 3h", Detalhe: "Aniversário Rita — " + contas.length + " contas abertas" });
    }
    alertas.push({ Severidade: "media", Titulo: "1 dívida com mais de 30 dias", Detalhe: clientes[0].Nome + " — 2 300,00 MT" });
    return {
      TemLoja: true, NegocioNome: L.nome, TipoNegocio: tipo,
      VendasHoje: hoje, VendasHojeVariacaoPct: Math.round(((hoje - ontem) / ontem) * 1000) / 10,
      LucroEstimadoHoje: Math.round(hoje * 0.28), MargemHojePct: 28,
      ContasAbertasAgora: tipo === "bottlestore" ? contas.length : 0, ContasAbertasValor: tipo === "bottlestore" ? emConsumo : 0,
      DividasEmAbertoValor: 5400, DividasClientes: 2, DividasAntigasCount: 1, ProdutosAbaixoMinimo: abaixo.length,
      SerieVendas14Dias: serie, Alertas: alertas,
      MaisVendidosHoje: catalogo.slice(0, 5).map(function (c, i) {
        return { ProdutoNome: c.Produto.Nome, Quantidade: 14 - i * 3 };
      }),
    };
  }

  var ADMIN = {
    PrecisaConfigurarAdmin: function () { return false; },
    ObterLojaAtual: function () { return loja; },
    Login: function () { return { ID: 1, Nome: "Administrador", Username: "admin" }; },
    CriarAdministrador: function () { return { ID: 1, Nome: "Administrador", Username: "admin" }; },
    ObterDashboard: dashboard,
    ListarCategorias: function () { return categorias; },
    ListarProdutos: function () { return produtos; },
    ListarTodasUnidades: function () { return unidadesTodas; },
    ListarUnidadesProduto: function (id) { return unidadesTodas.filter(function (u) { return u.ProdutoID === id; }); },
    ListarArmazens: function () { return armazens; },
    ListarStockPorArmazem: function (armazemId) {
      return produtos.filter(function (p) { return p.ControlaStock; }).map(function (p) {
        return {
          ProdutoID: p.ID, ArmazemID: armazemId, Quantidade: armazemId === 1 ? stock[p.ID] : Math.round(stock[p.ID] * 0.5),
          CustoMedioPonderado: unidadesTodas[p.ID - 1].PrecoCusto,
        };
      });
    },
    ListarMovimentosStock: function () { return []; },
    ListarClientes: function () {
      return clientes.map(function (c, i) {
        return {
          Cliente: c, TotalEmAberto: i === 0 ? 2300 : i === 1 ? 3100 : 0, DividasAbertas: i < 2 ? 1 : 0,
          DiasDividaMaisAntiga: i === 0 ? 41 : i === 1 ? 6 : 0, LimiteExcedido: false,
        };
      });
    },
    ListarVendas: function (desde, ate) {
      return todasVendas().filter(function (v) {
        var d = v.Venda.CriadoEm.slice(0, 10);
        return (!desde || d >= desde) && (!ate || d <= ate);
      }).reverse().map(function (v) { return { Venda: v.Venda, ClienteNome: v.ClienteNome, OperadorNome: v.OperadorNome }; });
    },
    ListarOperadores: function () { return operadores; },
    ListarInventarios: function () {
      return [
        { ID: 2, NegocioID: 1, ArmazemID: 1, Data: diaISO(3), Status: "aberto", UtilizadorID: 1 },
        { ID: 1, NegocioID: 1, ArmazemID: 1, Data: diaISO(33), Status: "fechado", UtilizadorID: 1 },
      ];
    },
    ListarFichasTecnicas: function () {
      if (tipo !== "bottlestore") return [];
      return catalogo.filter(function (c) { return c.Produto.TipoProduto === "composto"; }).map(function (c) {
        return {
          Produto: c.Produto, PrecoVenda: c.Unidades[0].PrecoVenda, CustoTeorico: Math.round(c.Unidades[0].PrecoVenda * 0.38),
          Disponivel: c.Disponivel,
          Ingredientes: [
            { ID: 1, ProdutoCompostoID: c.Produto.ID, ProdutoIngredienteID: 13, Quantidade: 0.3, PerdaPercentual: 5 },
          ],
        };
      });
    },
    ListarLotesAVencer: function () {
      return (L.lotes || []).map(function (l, i) {
        var d = new Date();
        d.setDate(d.getDate() + l[2]);
        return {
          ID: i + 1, ProdutoID: l[0], ArmazemID: 1, NumeroLote: l[1], DataValidade: d.toISOString(),
          Quantidade: l[3], CustoUnitario: unidadesTodas[l[0] - 1].PrecoCusto,
        };
      });
    },
    ListarVendasDoDia: function () { return vendasHoje().map(function (v) { return v.Venda; }); },
    CatalogoVenda: function () {
      return catalogo.map(function (c) { return { Produto: c.Produto, Unidades: c.Unidades, Stock: c.Disponivel }; });
    },
    AbrirPDV: function () {
      if (window.Nativo) window.Nativo.abrirPDV();
      return null;
    },
    ObterEstadoArranque: function () { return { PrecisaConfigurar: false, OperadoresAtivos: operadores.length }; },
    EstadoFaturacao: function () { return { Ativa: false }; },
  };

  var PDV = {
    EstaConfigurado: function () { return true; },
    ObterEstadoConfig: function () {
      return {
        Negocio: negocio, ModulosAtivos: L.modulos, Tema: tipo, Logo: "", PapelRecibo: "58", NUIT: "400123456",
        Endereco: "Av. 24 de Julho, Maputo", Telefone: "+258 84 000 0000", Email: "", TaxaIVA: 16,
      };
    },
    ObterOperadorAtual: function () {
      var op = operadores.filter(function (o) { return o.ID === operadorTurno; })[0];
      return op || { ID: 0, Nome: "", Ativo: false };
    },
    ListarOperadoresAtivos: function () { return operadores; },
    IniciarTurno: function (id, pin) {
      if (!/^[0-9]{4,6}$/.test(pin)) throw new Error("PIN errado");
      operadorTurno = id;
      try { sessionStorage.setItem("demo-turno", String(id)); } catch (e) {}
      return operadores.filter(function (o) { return o.ID === id; })[0];
    },
    TerminarTurno: function () {
      operadorTurno = 0;
      try { sessionStorage.removeItem("demo-turno"); } catch (e) {}
      return null;
    },
    ListarArmazens: function () { return armazens; },
    ListarClientes: function () { return clientes; },
    ListarCategorias: function () { return categorias; },
    ListarProdutos: function () { return produtos; },
    ListarVendasDoDia: function () { return vendasHoje().map(function (v) { return v.Venda; }); },
    CatalogoPDV: function () { return catalogo; },
    RegistarVendaDireta: function (p) {
      var itens = (p.Itens || []).map(function (it) { return itemRecibo(it.UnidadeProdutoID, it.Quantidade); });
      return { Venda: registarVendaPDV(itens, p.DescontoTotal, p.Pagamentos || [], p.ClienteID, "direta") };
    },
    ObterRecibo: function (id) {
      var v = vendasPDV.filter(function (x) { return x.resumo.Venda.ID === id; })[0];
      if (v) return v.recibo;
      var h = vendasHistorico.filter(function (x) { return x.Venda.ID === id; })[0];
      if (!h) return null;
      return recibo(id, h._itens.map(function (it) { return itemRecibo(it[0].Unidades[0].ID, it[1]); }), 0,
        [{ Metodo: "dinheiro", Valor: h.Venda.Total }], h.ClienteNome);
    },
    // --- Contas abertas ---
    ListarContasAbertas: function () { return contas.map(function (c) { return c.conta; }); },
    ListarItensConta: function (id) { var c = contaPorId(id); return c ? c.itens : []; },
    AbrirConta: function (req) {
      var id = contas.reduce(function (a, c) { return Math.max(a, c.conta.ID); }, 0) + 1;
      var conta = {
        ID: id, NegocioID: 1, ClienteID: req.ClienteID || 1, Identificador: req.Identificador, Status: "aberta",
        AbertaEm: new Date().toISOString(), FechadaEm: null, UtilizadorAberturaID: 1, UtilizadorFechoID: null, MotivoCancelamento: null,
      };
      contas.push({ conta: conta, itens: [] });
      gravarContas();
      return conta;
    },
    AdicionarItemConta: function (p) {
      var c = contaPorId(p.ContaID);
      var u = porUnidade[p.UnidadeProdutoID];
      // A mesma bebida soma-se à linha que já existe (como no backend).
      var igual = c.itens.filter(function (i) {
        return i.ProdutoID === p.ProdutoID && i.UnidadeProdutoID === p.UnidadeProdutoID && !i.Desconto;
      })[0];
      if (igual && !p.Desconto) {
        igual.Quantidade += p.Quantidade;
        gravarContas();
        return igual;
      }
      var item = {
        ID: proximoIdItem(), ContaID: p.ContaID, ProdutoID: p.ProdutoID, UnidadeProdutoID: p.UnidadeProdutoID,
        Quantidade: p.Quantidade, PrecoUnitarioAplicado: u.unidade.PrecoVenda, Desconto: 0,
        CriadoEm: new Date().toISOString(), UtilizadorID: 1,
      };
      c.itens.push(item);
      gravarContas();
      return item;
    },
    AlterarQuantidadeItemConta: function (itemId, quantidade) {
      var item = null;
      contas.forEach(function (c) {
        c.itens = c.itens.filter(function (i) {
          if (i.ID !== itemId) return true;
          i.Quantidade = quantidade;
          item = i;
          return quantidade > 0;
        });
      });
      gravarContas();
      return item;
    },
    TransferirItemConta: function (itemId, novaContaId) {
      var item = null;
      contas.forEach(function (c) {
        c.itens = c.itens.filter(function (i) { if (i.ID === itemId) { item = i; return false; } return true; });
      });
      if (item) { item.ContaID = novaContaId; contaPorId(novaContaId).itens.push(item); }
      gravarContas();
      return null;
    },
    CancelarConta: function (id) {
      var c = contaPorId(id);
      contas = contas.filter(function (x) { return x.conta.ID !== id; });
      gravarContas();
      return c ? c.conta : null;
    },
    FecharContaAberta: function (id, p) {
      var c = contaPorId(id);
      var itens = c.itens.map(function (i) { return itemRecibo(i.UnidadeProdutoID, i.Quantidade); });
      var venda = registarVendaPDV(itens, p.DescontoTotal, p.Pagamentos || [], c.conta.ClienteID, "conta_fechada");
      contas = contas.filter(function (x) { return x.conta.ID !== id; });
      gravarContas();
      return { Venda: { Venda: venda }, Alerta: { Excedido: false } };
    },
  };

  function responder(alvo, metodo, args) {
    var tabela = alvo === "App" ? PDV : ADMIN;
    if (Object.prototype.hasOwnProperty.call(tabela, metodo)) return copia(tabela[metodo].apply(null, args));
    if (/^(Listar|Relatorio|Catalogo|Vendas)/.test(metodo)) return [];
    return null;
  }

  function criarAlvo(nome) {
    return new Proxy({}, {
      get: function (_, metodo) {
        return function () {
          var args = Array.prototype.slice.call(arguments);
          console.log("[demo] " + nome + "." + String(metodo));
          return new Promise(function (resolve, reject) {
            setTimeout(function () {
              try { resolve(responder(nome, String(metodo), args)); } catch (e) { reject(String(e)); }
            }, 40);
          });
        };
      },
    });
  }

  window.go = { admin: { AdminApp: criarAlvo("AdminApp") }, api: { App: criarAlvo("App") } };

  // --- Seletor de loja (só no ecrã de login do admin) ---------------------
  function seletor() {
    var el = document.createElement("div");
    el.id = "__demo-seletor";
    el.style.cssText =
      "position:fixed;top:12px;left:12px;right:12px;z-index:9999;background:#14161c;color:#fff;" +
      "border-radius:12px;padding:10px 12px;font:13px/1.3 Roboto,sans-serif;box-shadow:0 6px 20px rgba(0,0,0,.25)";
    var topo = document.createElement("div");
    topo.style.cssText = "display:flex;align-items:center;margin-bottom:8px";
    var titulo = document.createElement("div");
    titulo.textContent = "Demonstração · tipo de loja";
    titulo.style.cssText = "opacity:.7;flex:1";
    topo.appendChild(titulo);
    el.appendChild(topo);
    var linha = document.createElement("div");
    linha.style.cssText = "display:flex;margin:-3px";
    TIPOS.forEach(function (t) {
      var b = document.createElement("button");
      b.textContent = ROTULO[t];
      var ativo = t === tipo;
      b.style.cssText =
        "flex:1 1 0;min-width:0;margin:3px;padding:10px 2px;border-radius:8px;font:600 12.5px Roboto,sans-serif;border:1px solid " +
        (ativo ? "#fff" : "#3a3f4b") + ";background:" + (ativo ? "#fff" : "transparent") + ";color:" + (ativo ? "#14161c" : "#fff");
      b.onclick = function () {
        gravar("demo:tipo", t);
        location.reload();
      };
      linha.appendChild(b);
    });
    el.appendChild(linha);
    var limpar = document.createElement("button");
    limpar.textContent = "Repor dados";
    limpar.style.cssText = "background:none;border:none;color:#9aa1ad;font:12px Roboto,sans-serif;padding:0;text-decoration:underline";
    limpar.onclick = function () {
      TIPOS.forEach(function (t) { gravar("demo:" + t + ":vendas", []); try { localStorage.removeItem("demo:" + t + ":contas"); } catch (e) { /* */ } });
      location.reload();
    };
    topo.appendChild(limpar);
    return el;
  }
  if (/\/admin\//.test(location.pathname)) {
    var painel = null;
    setInterval(function () {
      var noLogin = !!document.querySelector(".login-shell");
      if (noLogin && !painel) { painel = seletor(); document.body.appendChild(painel); }
      if (!noLogin && painel) { painel.parentNode.removeChild(painel); painel = null; }
    }, 400);
  }
})();
