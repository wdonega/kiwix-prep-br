"use strict";

const $ = (id) => document.getElementById(id);
const ico = (nome, cls = "") => `<svg class="i ${cls}" aria-hidden="true"><use href="/prep/icones.svg#i-${nome}"/></svg>`;
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

let est = null;                                   // último /api/estado
let aba = sessionStorage.getItem("prep.aba") || "todos";
const travados = new Set();                       // ids com clique em andamento (evita clique duplo)
let htmlLista = "", htmlAbas = "";                // só troca o DOM quando o desenho muda de verdade

// ---------------------------------------------------------------- texto e números

function tamanho(b) {
  if (b == null) return "–";
  const k = 1000;
  if (b >= k ** 3) return (b / k ** 3).toLocaleString("pt-BR", { minimumFractionDigits: 1, maximumFractionDigits: 1 }) + " GB";
  if (b >= k ** 2) return Math.round(b / k ** 2) + " MB";
  return Math.max(1, Math.round(b / k)) + " kB";
}

const dataValida = (iso) => iso && !iso.startsWith("0001");

function quando(iso) {
  if (!dataValida(iso)) return null;
  const d = new Date(iso), hoje = new Date();
  const hora = d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
  if (d.toDateString() === hoje.toDateString()) return `hoje, ${hora}`;
  return `${d.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" })}, ${hora}`;
}

const dia = (iso) => new Date(iso).toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric" });

function falta(seg) {
  if (seg >= 3600) return `${Math.floor(seg / 3600)} h ${Math.round((seg % 3600) / 60)} min`;
  if (seg >= 60) return `${Math.round(seg / 60)} min`;
  return `${Math.max(1, Math.round(seg))} s`;
}

// ---------------------------------------------------------------- servidor

async function api(metodo, caminho) {
  const r = await fetch(caminho, { method: metodo, headers: { "X-Prep-Admin": "1" }, cache: "no-store" });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.erro || `HTTP ${r.status}`);
  return j;
}

async function acao(tipo, id) {
  travados.add(id);
  render();
  try {
    await api("POST", `/api/${tipo}/${encodeURIComponent(id)}`);
  } catch (e) {
    avisar(e.message.charAt(0).toUpperCase() + e.message.slice(1) + ".");
  }
  travados.delete(id);
  await atualizar();
}

// ---------------------------------------------------------------- avisos e confirmação

function avisar(msg, tipo = "erro") {
  const el = document.createElement("div");
  el.className = "aviso-f " + tipo;
  el.innerHTML = ico(tipo === "erro" ? "alert" : "check-circle") + `<span>${esc(msg)}</span>`;
  $("avisos").append(el);
  setTimeout(() => el.remove(), 6000);
}

function confirmar({ titulo, texto, botao, icone = "trash" }) {
  const dlg = $("dlg");
  $("dlg-ico").innerHTML = ico(icone);
  $("dlg-tit").textContent = titulo;
  $("dlg-txt").textContent = texto;
  $("dlg-sim").textContent = botao;
  dlg.returnValue = "";
  dlg.showModal();
  return new Promise((ok) => dlg.addEventListener("close", () => ok(dlg.returnValue === "sim"), { once: true }));
}

// ---------------------------------------------------------------- desenho

const EM_ANDAMENTO = ["na_fila", "baixando", "verificando", "apagando"];
const ICONE = { instalado: "check-circle", desatualizado: "arrow-up-circle", nao_instalado: "cloud-download",
  na_fila: "clock", baixando: "download", verificando: "check", apagando: "trash" };
const ETIQUETA = { desatualizado: ["versão nova", "etq-aviso"], na_fila: ["na fila", "etq-info"], baixando: ["baixando", "etq-info"],
  verificando: ["conferindo", "etq-info"], apagando: ["apagando", "etq-info"] };

const ABAS = [
  ["todos", "Todos", () => true],
  ["instalados", "Instalados", (it) => !!it.instalado],
  ["disponiveis", "Disponíveis", (it) => !it.instalado],
  ["atualizacoes", "Atualizações", (it) => !!it.instalado && (it.situacao === "desatualizado" || ["na_fila", "baixando", "verificando"].includes(it.situacao))],
];

function linha(it) {
  const s = it.situacao, inst = it.instalado, disp = it.disponivel;
  const trava = travados.has(it.id) ? "disabled" : "";
  const id = esc(it.id);

  const meta = [];
  if (inst) {
    meta.push("Instalado", inst.edicao ? `edição ${esc(inst.edicao)}` : "edição desconhecida");
    if (dataValida(inst.baixado_em)) meta.push(`baixado em ${dia(inst.baixado_em)}`);
    if (s === "desatualizado" && disp) meta.push(`<span class="novo">nova: edição ${esc(disp.edicao)} · ${tamanho(disp.tamanho)}</span>`);
  } else if (disp) {
    meta.push(`Edição ${esc(disp.edicao)}`);
  } else {
    meta.push(it.erro_verificacao ? "Não foi possível verificar a versão" : "Versão ainda não verificada");
  }

  const tam = inst ? inst.tamanho : disp ? disp.tamanho : null;
  if (tam != null) meta.splice(inst ? 1 : 0, 0, `<span class="so-cel">${tamanho(tam)}</span>`);
  const txtTam = tam != null ? `<span class="lin-tam">${tamanho(tam)}</span>` : "";
  const apagar = `<button class="ibtn perigo" type="button" data-a="apagar" data-id="${id}" data-dica="Apagar" aria-label="Apagar ${esc(it.titulo)}" ${trava}>${ico("trash")}</button>`;
  let acoes = "";
  if (s === "nao_instalado") {
    acoes = `${txtTam}<button class="btn btn-suave" type="button" data-a="baixar" data-id="${id}" aria-label="Baixar ${esc(it.titulo)}" ${trava}>${ico("download")}<span class="rot">Baixar</span></button>`;
  } else if (s === "instalado") {
    acoes = txtTam + apagar;
  } else if (s === "desatualizado") {
    acoes = `${txtTam}<button class="btn btn-aviso" type="button" data-a="baixar" data-id="${id}" aria-label="Atualizar ${esc(it.titulo)}" ${trava}>${ico("refresh")}<span class="rot">Atualizar</span></button>${apagar}`;
  } else if (s === "na_fila" || s === "baixando") {
    acoes = `<button class="btn" type="button" data-a="cancelar" data-id="${id}" aria-label="Cancelar o download de ${esc(it.titulo)}" ${trava}>${ico("x")}<span class="rot">Cancelar</span></button>`;
  }

  let extra = "";
  if (s === "baixando" || s === "verificando") {
    extra += `<div class="prog" data-prog="${id}"><div class="prog-barra"><i></i></div>
      <div class="prog-info"><span data-p="esq"></span><span data-p="dir"></span></div></div>`;
  } else if (s === "na_fila") {
    extra += `<div class="prog"><div class="prog-info"><span>Aguardando o download anterior terminar…</span></div></div>`;
  }
  if (it.erro) extra += `<div class="lin-erro">${ico("alert")}<span>${esc(it.erro)}</span></div>`;

  const etq = ETIQUETA[s] ? `<span class="etq ${ETIQUETA[s][1]}">${ETIQUETA[s][0]}</span>` : "";
  return `<div class="lin s-${s}">
    <span class="lin-st">${ico(ICONE[s] || "package", s === "verificando" ? "" : "")}</span>
    <div class="lin-corpo">
      <div class="lin-tit">${esc(it.titulo)}${etq}</div>
      <div class="lin-desc">${esc(it.descricao)}</div>
      <div class="lin-meta">${meta.map((m) => (m.startsWith("<span") ? m : `<span>${m}</span>`)).join("")}</div>
    </div>
    <div class="lin-acoes">${acoes}</div>
    ${extra ? `<div class="lin-extra">${extra}</div>` : ""}
  </div>`;
}

// linhas de um grupo, com um subtítulo cada vez que o tema muda (grupos grandes, como o Prepper)
function comTemas(itens) {
  let html = "", tema = "";
  for (const it of itens) {
    if ((it.tema || "") !== tema) {
      tema = it.tema || "";
      if (tema) html += `<div class="lin-tema">${esc(tema)}</div>`;
    }
    html += linha(it);
  }
  return html;
}

// os números do progresso mudam a cada segundo: atualiza só eles, sem redesenhar a lista
function atualizarProgresso(it) {
  const el = document.querySelector(`[data-prog="${CSS.escape(it.id)}"]`);
  const p = it.progresso;
  if (!el || !p) return;
  const conferindo = it.situacao === "verificando";
  const pct = p.total > 0 ? Math.min(100, (p.baixado / p.total) * 100) : 0;
  el.querySelector(".prog-barra").classList.toggle("indefinida", conferindo);
  el.querySelector(".prog-barra i").style.width = pct + "%";
  let esq, dir = "";
  if (conferindo) {
    esq = "Conferindo a integridade do arquivo (sha-256)…";
  } else {
    esq = `<b>${Math.floor(pct)}%</b> · ${tamanho(p.baixado)} de ${tamanho(p.total)}`;
    const partes = [];
    if (p.velocidade > 0) {
      partes.push(`${tamanho(p.velocidade)}/s`);
      if (p.total > p.baixado) partes.push(`faltam ${falta((p.total - p.baixado) / p.velocidade)}`);
    }
    if (p.mirror) partes.push(esc(p.mirror));
    dir = partes.join(" · ");
  }
  el.querySelector('[data-p="esq"]').innerHTML = esq;
  el.querySelector('[data-p="dir"]').innerHTML = dir;
}

function render() {
  if (!est) return;
  const itens = est.itens;
  const instalados = itens.filter((i) => i.instalado);
  const emUso = instalados.reduce((s, i) => s + (i.instalado.tamanho || 0), 0);

  // resumo
  $("s-instalados").innerHTML = `${instalados.length} <small>de ${itens.length}</small>`;
  $("s-ocupado").textContent = tamanho(emUso);
  if (est.disco) {
    const { livre, total } = est.disco;
    const outros = Math.max(0, total - livre - emUso);
    $("s-livre").textContent = `em uso · ${tamanho(livre)} livres`;
    $("s-disco-outros").style.width = (outros / total) * 100 + "%";
    $("s-disco-acervo").style.width = Math.max(emUso > 0 ? 0.6 : 0, (emUso / total) * 100) + "%";
    $("s-disco-rot").textContent = `disco de ${tamanho(total)}`;
  } else {
    $("s-livre").textContent = "em uso";
  }
  $("s-verificado").textContent = est.verificando ? "verificando…" : quando(est.verificado_em) || "ainda não";
  $("b-verificar").disabled = est.verificando;
  $("b-verificar").querySelector(".i").classList.toggle("girando", est.verificando);
  $("s-catalogo").textContent = `Catálogo v${est.catalogo_versao}`;

  const falhas = itens.filter((i) => i.erro_verificacao).length;
  $("alerta").hidden = est.verificando || falhas === 0;
  if (falhas) {
    $("alerta").innerHTML = ico("alert") + `<span>Não consegui verificar a versão mais recente de ${falhas} ${falhas === 1 ? "acervo" : "acervos"}
      (sem internet?). Mostrando a última informação conhecida.</span>`;
  }

  // abas
  const contagens = Object.fromEntries(ABAS.map(([id, , f]) => [id, itens.filter(f).length]));
  if (aba === "atualizacoes" && !contagens.atualizacoes) aba = "todos";
  const abas = ABAS.filter(([id]) => id !== "atualizacoes" || contagens.atualizacoes > 0)
    .map(([id, rot]) => `<button class="aba ${id === "atualizacoes" ? "aviso" : ""}" type="button" role="tab" aria-selected="${aba === id}" data-aba="${id}">
      ${rot}<span class="n">${contagens[id]}</span></button>`).join("");
  if (abas !== htmlAbas) $("abas").innerHTML = htmlAbas = abas;

  // lista
  const filtro = ABAS.find(([id]) => id === aba)[2];
  let html = "";
  for (const g of est.grupos) {
    const doGrupo = itens.filter((i) => i.grupo === g.id);
    const visiveis = doGrupo.filter(filtro);
    if (!visiveis.length) continue;
    const n = doGrupo.filter((i) => i.instalado).length;
    html += `<section class="grupo">
      <div class="grupo-cab"><h2>${esc(g.titulo)}</h2>${g.dica ? `<span class="etq">${esc(g.dica)}</span>` : ""}
        <span class="cont">${n} de ${doGrupo.length} ${n === 1 ? "instalado" : "instalados"}</span></div>
      <div class="painel-lista">${comTemas(visiveis)}</div></section>`;
  }
  if (!html) {
    html = `<p class="nada">${aba === "instalados" ? "Nenhum acervo instalado ainda. Escolha um na aba “Disponíveis”." : "Nada por aqui."}</p>`;
  }
  if (html !== htmlLista) $("lista").innerHTML = htmlLista = html;
  itens.forEach(atualizarProgresso);
}

// ---------------------------------------------------------------- ciclo

let timer = null, falhasSeguidas = 0;
async function atualizar() {
  clearTimeout(timer);
  try {
    est = await api("GET", "/api/estado");
    falhasSeguidas = 0;
    render();
  } catch (e) {
    if (++falhasSeguidas === 3) avisar("Sem resposta do gerenciador. Tentando de novo…");
  }
  const ativo = est && (est.verificando || est.itens.some((i) => EM_ANDAMENTO.includes(i.situacao)));
  timer = setTimeout(atualizar, ativo ? 1000 : 5000);
}

document.addEventListener("click", async (ev) => {
  const tab = ev.target.closest("[data-aba]");
  if (tab) {
    aba = tab.dataset.aba;
    sessionStorage.setItem("prep.aba", aba);
    render();
    return;
  }
  const b = ev.target.closest("[data-a]");
  if (!b || !est) return;
  const id = b.dataset.id, a = b.dataset.a;
  const it = est.itens.find((i) => i.id === id);
  if (!it) return;
  if (a === "apagar") {
    const ok = await confirmar({ titulo: `Apagar “${it.titulo}”?`, botao: "Apagar",
      texto: `Libera ${tamanho(it.instalado?.tamanho)} em disco. Para ter de volta é só baixar de novo.` });
    if (ok) acao("apagar", id);
  } else if (a === "cancelar") {
    const ok = await confirmar({ titulo: "Cancelar o download?", botao: "Cancelar download", icone: "x",
      texto: `O que já foi baixado de “${it.titulo}” é descartado.` });
    if (ok) acao("cancelar", id);
  } else {
    acao(a, id);
  }
});

$("b-verificar").addEventListener("click", async () => {
  $("b-verificar").disabled = true;
  try { await api("POST", "/api/verificar"); } catch (e) { avisar(e.message); }
  setTimeout(atualizar, 300);
});

document.addEventListener("visibilitychange", () => { if (!document.hidden) atualizar(); });
atualizar();
