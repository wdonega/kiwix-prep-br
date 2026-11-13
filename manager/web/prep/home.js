"use strict";

const $ = (id) => document.getElementById(id);
const ico = (nome, cls = "") => `<svg class="i ${cls}" aria-hidden="true"><use href="/prep/icones.svg#i-${nome}"/></svg>`;

let livros = null;   // null = ainda carregando
let grupos = [];
let grupoSel = sessionStorage.getItem("prep.grupo") || "";   // "" = todos
let consulta = "";

// ---------------------------------------------------------------- texto e números

function tamanho(b) {
  const k = 1000;
  if (b >= k ** 3) return (b / k ** 3).toLocaleString("pt-BR", { maximumFractionDigits: 1 }) + " GB";
  if (b >= k ** 2) return Math.round(b / k ** 2) + " MB";
  return Math.max(1, Math.round(b / k)) + " kB";
}

function contagem(n, um, varios) {
  const txt = n >= 100000
    ? n.toLocaleString("pt-BR", { notation: "compact", maximumFractionDigits: 1 })
    : n.toLocaleString("pt-BR");
  return `${txt} ${n === 1 ? um : varios}`;
}

const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const semAcento = (s) => s.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();

// destaca o trecho buscado, comparando sem acento ("agua" acha "Água")
function destacar(texto, q) {
  if (!q) return esc(texto);
  const i = semAcento(texto).indexOf(q);
  if (i < 0 || semAcento(texto).length !== texto.length) return esc(texto);
  return esc(texto.slice(0, i)) + "<mark>" + esc(texto.slice(i, i + q.length)) + "</mark>" + esc(texto.slice(i + q.length));
}

// ---------------------------------------------------------------- desenho

function card(l, q) {
  const grupo = grupos.find((g) => g.id === l.grupo);
  const sub = [l.tema || (grupo && grupo.titulo), l.edicao && `edição ${l.edicao}`].filter(Boolean).join(" · ");
  // etiqueta de idioma só no que não é em português (o resto do acervo é)
  const outros = l.idiomas.includes("por") ? [] : l.idiomas;
  const info = [];
  if (l.artigos > 0) info.push(contagem(l.artigos, "artigo", "artigos"));
  info.push(tamanho(l.tamanho));
  return `<a class="card" href="/viewer#${encodeURIComponent(l.nome)}">
    <div class="card-topo">
      <span class="card-ico"><img src="${esc(l.icone)}" alt="" width="38" height="38" loading="lazy"></span>
      <div>
        <div class="card-tit">${destacar(l.titulo, q)}</div>
        ${sub ? `<div class="card-sub">${esc(sub)}</div>` : ""}
      </div>
    </div>
    <div class="card-desc">${destacar(l.descricao, q)}</div>
    <div class="card-rodape">
      ${outros.map((c) => `<span class="etq" title="Em ${esc(nomeIdioma(c))}">${esc(SIGLAS_IDIOMA[c] || c.toUpperCase())}</span>`).join("")}
      ${info.map((t) => `<span class="card-info">${esc(t)}</span>`).join("")}
      ${ico("arrow-right", "card-seta")}
    </div>
  </a>`;
}

const NOMES_IDIOMA = { por: "português", eng: "inglês", spa: "espanhol", fra: "francês", deu: "alemão", ita: "italiano" };
const SIGLAS_IDIOMA = { por: "PT", eng: "EN", spa: "ES", fra: "FR", deu: "DE", ita: "IT" };
function nomeIdioma(cod) {
  if (NOMES_IDIOMA[cod]) return NOMES_IDIOMA[cod];
  try { return new Intl.DisplayNames(["pt-BR"], { type: "language" }).of(cod) || cod; } catch (e) { return cod; }
}

// A página mostra sempre todos os acervos, de qualquer idioma. Só a busca dentro do texto tem um limite, que é
// do Kiwix: ele recusa buscar em acervos de idiomas diferentes ao mesmo tempo (e em acervos com mais de um idioma).
// Por isso a busca é oferecida por idioma: português primeiro, depois os outros, do que tem mais acervos para o que tem menos.
function buscasPorIdioma() {
  const porIdioma = new Map();
  for (const l of livros || []) {
    if (l.idiomas.length !== 1) continue;
    if (!porIdioma.has(l.idiomas[0])) porIdioma.set(l.idiomas[0], []);
    porIdioma.get(l.idiomas[0]).push(l);
  }
  return [...porIdioma.entries()]
    .map(([idioma, alvos]) => ({ idioma, alvos }))
    .sort((a, b) => (b.idioma === "por") - (a.idioma === "por") || b.alvos.length - a.alvos.length);
}

function urlBuscaDentro(alvos) {
  return "/viewer#search?pattern=" + encodeURIComponent(consulta.trim()) +
    alvos.map((l) => "&books.name=" + encodeURIComponent(l.nome)).join("");
}

function render() {
  if (livros === null) {
    $("lista").innerHTML = `<div class="grade">${'<div class="esqueleto"></div>'.repeat(3)}</div>`;
    return;
  }
  const q = semAcento(consulta.trim());
  const achados = livros.filter((l) => !q || semAcento(l.titulo + " " + l.descricao).includes(q));
  const comLivros = grupos.filter((g) => achados.some((l) => l.grupo === g.id));
  if (grupoSel && !comLivros.some((g) => g.id === grupoSel)) grupoSel = "";        // a categoria ficou vazia
  const visiveis = achados.filter((l) => !grupoSel || l.grupo === grupoSel);

  // resumo
  const total = livros.reduce((s, l) => s + l.tamanho, 0);
  $("resumo").textContent = livros.length ? `${contagem(livros.length, "acervo", "acervos")} · ${tamanho(total)}` : "Nenhum acervo instalado";

  // categorias (só quando há mais de uma)
  let filtros = "";
  if (comLivros.length > 1) {
    const chip = (id, rot, n) => `<button class="chip" type="button" aria-pressed="${grupoSel === id}" data-grupo="${esc(id)}">${esc(rot)}<span class="n">${n}</span></button>`;
    filtros += chip("", "Todos", achados.length) + comLivros.map((g) => chip(g.id, g.titulo, achados.filter((l) => l.grupo === g.id).length)).join("");
  }
  $("filtros").hidden = !filtros;
  $("filtros").innerHTML = filtros;

  let html = "";
  const buscas = q ? buscasPorIdioma() : [];
  const alvos = buscas.length ? buscas[0].alvos : [];
  buscas.forEach((b, i) => {
    const onde = buscas.length > 1 ? `nos acervos em ${nomeIdioma(b.idioma)}` : "dentro dos acervos";
    html += `<a class="buscar-dentro" href="${esc(urlBuscaDentro(b.alvos))}">
      ${ico("search")}<span>Buscar <b>“${esc(consulta.trim())}”</b> ${esc(onde)}</span>
      <span class="fim"><span>no texto de ${contagem(b.alvos.length, "acervo", "acervos")}</span>${i === 0 ? "<kbd>Enter</kbd>" : ""}</span></a>`;
  });

  if (!livros.length) {
    html += `<div class="vazio"><div class="vazio-ico">${ico("library")}</div>
      <h2>Nenhum acervo instalado ainda</h2>
      <p>Escolha o que baixar no gerenciador. Depois de baixado, tudo funciona sem internet.</p>
      <a class="btn btn-pri" href="/admin/">${ico("settings")}Abrir o gerenciador</a></div>`;
  } else if (!visiveis.length) {
    html += `<div class="vazio"><div class="vazio-ico">${ico("search")}</div>
      <h2>Nenhum acervo com esse nome</h2>
      <p>${alvos.length ? "A busca acima procura dentro do texto dos acervos." : "Tente outro termo."}</p></div>`;
  } else {
    html += `<div class="grade">${visiveis.map((l) => card(l, q)).join("")}</div>`;
  }
  $("lista").innerHTML = html;
}

document.addEventListener("click", (ev) => {
  const chip = ev.target.closest("[data-grupo]");
  if (chip) {
    grupoSel = chip.dataset.grupo;
    sessionStorage.setItem("prep.grupo", grupoSel);
    render();
  }
});

// ---------------------------------------------------------------- busca

$("q").addEventListener("input", () => { consulta = $("q").value; render(); });
$("busca").addEventListener("submit", (ev) => {
  ev.preventDefault();
  const buscas = consulta.trim() ? buscasPorIdioma() : [];
  if (buscas.length) location.href = urlBuscaDentro(buscas[0].alvos);
});
document.addEventListener("keydown", (ev) => {
  if (ev.key === "Escape") {
    if (document.activeElement === $("q") && consulta) { $("q").value = consulta = ""; render(); }
    return;
  }
  const digitando = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
  if (ev.key === "/" && !digitando && !ev.ctrlKey && !ev.metaKey && !ev.altKey) { ev.preventDefault(); $("q").focus(); }
});

// ---------------------------------------------------------------- dados

let ultima = "";
async function carregar() {
  try {
    const r = await fetch("/api/biblioteca", { cache: "no-store" });
    if (!r.ok) throw new Error("HTTP " + r.status);
    const texto = await r.text();
    if (texto === ultima) return; // nada mudou: não redesenha (preserva foco e rolagem)
    ultima = texto;
    const j = JSON.parse(texto);
    livros = j.livros; grupos = j.grupos;
    render();
  } catch (e) {
    if (livros === null) {
      $("resumo").textContent = "Sem resposta do servidor";
      $("lista").innerHTML = `<div class="vazio"><div class="vazio-ico">${ico("alert")}</div>
        <h2>Não consegui carregar a biblioteca</h2><p>Tentando de novo em alguns segundos…</p></div>`;
    }
  }
}

// /?q=texto abre a página já com a busca preenchida
const qInicial = new URLSearchParams(location.search).get("q");
if (qInicial) $("q").value = consulta = qInicial;

render();
carregar();
// um download que termina no gerenciador aparece aqui sem recarregar a página
setInterval(() => { if (!document.hidden) carregar(); }, 15000);
document.addEventListener("visibilitychange", () => { if (!document.hidden) carregar(); });
