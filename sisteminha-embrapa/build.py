#!/usr/bin/env python3
"""Gera sisteminha-embrapa_pt.zim a partir de conteudo/*.md + PDFs.

Uso:  python build.py [--html-dir DIR]   (requer: libzim, pymupdf, markdown, pillow)
"""
import datetime
import hashlib
import html
import io
import json
import os
import re
import sys
from pathlib import Path

import markdown
import pymupdf
from PIL import Image
from libzim.writer import Creator, Hint, Item, StringProvider

AQUI = Path(__file__).resolve().parent
CONTEUDO = AQUI / "conteudo"
ASSETS = AQUI / "assets"
# nome com hash do conteúdo: quando o CSS muda, a URL muda e o navegador não usa a versão velha do cache
CSS = f"assets/estilo-{hashlib.sha1((ASSETS / 'estilo.css').read_bytes()).hexdigest()[:8]}.css"
SAIDA = Path(os.environ.get("SAIDA_ZIM", AQUI / f"sisteminha-embrapa_pt_{datetime.date.today():%Y-%m}.zim"))

DOCS = {  # chave -> (arquivo, título curto, rótulo curto p/ figuras)
    "doc259": (AQUI / "fontes/SisteminhaEmbrapaUFUFapemigBaixa2019.pdf",
               "Documentos 259 – Módulo 1: tanque de peixes (2019)", "Documentos 259"),
    "doc122": (AQUI / "fontes/39.pdf", "Ficha Sisteminha – Tecnologias para a Agricultura Familiar (2024)", "Documentos 122"),
    "folder2024": (AQUI / "fontes/Folder-Sisteminha-Comunidades-Embrapa-Cocais-21-mar-2024.pdf",
                   "Folder Sisteminha Comunidades – Embrapa Cocais (2024)", "folder Sisteminha Comunidades"),
    "folder2013": (AQUI / "fontes/folder-2013-sistema-integrado-alternativo.pdf",
                   "Folder Sistema integrado alternativo – Agricultura familiar (2013)", "folder Embrapa Meio-Norte 2013"),
    "folder2012": (AQUI / "fontes/folder-2012-manutencao-motobombas.pdf",
                   "Folder Manutenção das motobombas submersas (2012)", "folder Embrapa Meio-Norte 2012"),
    "it029": (AQUI / "fontes/instrucao-tecnica-029-poedeira-embrapa-051.pdf",
              "Instrução Técnica 29 – Poedeira Embrapa 051 (2006)", "Instrução Técnica 29"),
}

# Créditos exibidos no topo de cada página: chave -> (referência, URL original)
CREDITOS = {
    "doc259": ("GUILHERME, L. C.; SOBREIRA, R. dos S.; OLIVEIRA, V. Q. de. <i>Sisteminha Embrapa - UFU - FAPEMIG: "
               "Sistema Integrado de Produção de Alimentos – Módulo 1: tanque de peixes</i>. Teresina: Embrapa "
               "Meio-Norte, 2019. 63 p. (Documentos, 259). Editora técnica: Magda Cruciol.",
               "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/1111791/1/SisteminhaEmbrapaUFUFapemigBaixa2019.pdf"),
    "doc122": ("RODRIGUES, L. A.; GUILHERME, L. C.; INOUE, L. A. K.; SILVA, T. S. de C. <i>Sisteminha "
               "Embrapa/UFU/Fapemig</i>. In: Tecnologias para a Agricultura Familiar. 4. ed. Embrapa, 2024. "
               "p. 235–238 (Documentos, 122).",
               "https://ainfo.cnptia.embrapa.br/digital/bitstream/doc/1169473/1/39.pdf"),
    "folder2024": ("EMBRAPA COCAIS. <i>Sisteminha Comunidades: desenvolvimento socioeconômico com "
                   "sustentabilidade</i>. Folder. São Luís, mar. 2024.",
                   "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/1163078/1/Folder-Sisteminha-Comunidades-Embrapa-Cocais-21-mar-2024.pdf"),
    "folder2013": ("EMBRAPA MEIO-NORTE. <i>Sistema integrado alternativo para produção de alimentos: agricultura "
                   "familiar</i>. Folder. Parnaíba, mar. 2013. Equipe: L. C. Guilherme, J. M. Kimpara, L. A. Rodrigues et al.",
                   "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/958146/1/FolAgriculturaFamiliar.pdf"),
    "folder2012": ("EMBRAPA MEIO-NORTE. <i>Sistema integrado alternativo para produção de alimentos: agricultura "
                   "familiar – manutenção das motobombas submersas</i>. Folder. Parnaíba, nov. 2012.",
                   "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/959514/1/agricfamiliarsistintegradobombas.pdf"),
    "it029": ("AVILA, V. S. de; JAENISCH, F. R. F.; FIGUEIREDO, E. A. P. de; SCHMIDT, G. S.; ROSA, P. S.; BRUM, "
              "P. A. R. de. <i>Sistema para produção de ovos com a poedeira Embrapa 051</i>. Concórdia: Embrapa "
              "Suínos e Aves, 2006 (Instrução Técnica para o Avicultor, 29).",
              "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/443750/1/publicacaop8f69k5x.pdf"),
}


def bloco_credito(chaves, extra=""):
    if not chaves and not extra:
        return ""
    itens = []
    for k in chaves:
        ref, url = CREDITOS[k]
        itens.append(f'<li>{ref} <a href="/pdf/{DOCS[k][0].name}">PDF original (cópia offline)</a> · '
                     f'<a href="{url}">na Embrapa</a> <small>(online)</small></li>')
    if extra:
        itens.append(f"<li>{extra}</li>")
    return (f'<div class="credito"><b>Fonte e créditos</b><ul>{"".join(itens)}</ul>'
            f'<small>Texto reorganizado a partir do original; números e recomendações mantidos. '
            f'Detalhes em <a href="/fontes">Fontes</a>.</small></div>')


def com_credito(corpo, bloco):
    if not bloco:
        return corpo
    i = corpo.find("</h1>")
    return corpo[:i + 5] + bloco + corpo[i + 5:] if i >= 0 else bloco + corpo


# Ordem do sumário: (seção, [caminhos])
SUMARIO = [
    ("Início", ["index"]),
    ("Visão geral", ["o-que-e", "principios", "modulos", "historico", "comunidades"]),
    ("Tanque de peixes – Perguntas e respostas", [
        "faq", "tanque/apresentacao", "tanque/vantagens", "tanque/construcao", "tanque/alimentacao",
        "tanque/funcionamento", "tanque/sanidade", "tanque/biofiltro", "tanque/sedimentador",
        "tanque/custos", "tanque/materiais"]),
    ("Manutenção e outros módulos", ["motobombas", "galinhas", "compostagem-minhocario", "horta"]),
    ("Ferramentas", ["calculadora-racao", "glossario", "busca"]),
    ("Fontes", ["fontes"]),
]
TITULO_CURTO = {
    "index": "Início", "o-que-e": "O que é o Sisteminha", "principios": "Princípios",
    "modulos": "Módulos de produção", "historico": "Histórico e resultados", "comunidades": "Sisteminha Comunidades",
    "faq": "Índice das 52 perguntas", "tanque/apresentacao": "1–7 · Apresentação",
    "tanque/vantagens": "8–10 · Água e resíduos", "tanque/construcao": "11–19 · Construção do tanque",
    "tanque/alimentacao": "20–21 · Alimentação", "tanque/funcionamento": "22–27 · Funcionamento",
    "tanque/sanidade": "28–30 · Controle sanitário", "tanque/biofiltro": "31–34 · Biofiltro",
    "tanque/sedimentador": "35–42 · Sedimentador", "tanque/custos": "43–47 · Custos",
    "tanque/materiais": "48–52 · Materiais", "motobombas": "Manutenção das motobombas",
    "galinhas": "Galinhas de postura", "compostagem-minhocario": "Compostagem e minhocário",
    "horta": "Horta e hidroponia", "calculadora-racao": "Calculadora de ração", "glossario": "Glossário",
    "busca": "Buscar", "fontes": "Fontes e documentos",
}

LIMITE_PDF = 6_000_000  # bytes

MD_EXT = ["tables", "attr_list", "def_list", "admonition", "md_in_html", "fenced_code", "sane_lists", "toc"]


# ---------------------------------------------------------------- utilidades
def ler_md(p: Path):
    s = p.read_text(encoding="utf-8")
    meta = {}
    if s.startswith("---\n"):
        cab, s = s[4:].split("\n---\n", 1)
        for linha in cab.splitlines():
            if ":" in linha:
                k, v = linha.split(":", 1)
                meta[k.strip()] = v.strip()
    return meta, s


def relativizar(html_txt: str, caminho: str) -> str:
    """Converte href/src "/x" em caminhos relativos ao artigo atual (ZIM não tem raiz absoluta)."""
    prof = caminho.count("/")
    pref = "../" * prof if prof else ""

    def sub(m):
        attr, alvo = m.group(1), m.group(2)
        if alvo.startswith("//"):
            return m.group(0)
        alvo = alvo[1:]
        if alvo == "" or alvo.startswith("#"):
            alvo = "index" + alvo
        return f'{attr}="{pref}{alvo}"'

    return re.sub(r'(href|src)="(/[^"]*)"', sub, html_txt)


def texto_puro(html_txt: str) -> str:
    t = re.sub(r"<(script|style)[^>]*>.*?</\1>", " ", html_txt, flags=re.S)
    t = re.sub(r"<[^>]+>", " ", t)
    return re.sub(r"\s+", " ", html.unescape(t)).strip()


# ---------------------------------------------------------------- PDFs -> imagens
class Midia:
    def __init__(self):
        self.arquivos = {}  # caminho -> (bytes, mimetype)
        self.compactado = set()
        self._abertos = {}

    def pdfs(self):
        for chave, (arq, _, _) in DOCS.items():
            dados = arq.read_bytes()
            if len(dados) > LIMITE_PDF:  # PDF só-imagem gigante: recria a partir de páginas em JPEG
                d = pymupdf.open(arq)
                leve = pymupdf.open()
                for pg in d:
                    zoom = 2400 / pg.rect.width
                    pix = pg.get_pixmap(matrix=pymupdf.Matrix(zoom, zoom), alpha=False)
                    im = Image.frombytes("RGB", (pix.width, pix.height), pix.samples)
                    nova = leve.new_page(width=pg.rect.width, height=pg.rect.height)
                    nova.insert_image(nova.rect, stream=self._jpg(im, 72))
                dados = leve.tobytes(garbage=3, deflate=True)
                self.compactado.add(chave)
            self.arquivos[f"pdf/{arq.name}"] = (dados, "application/pdf")

    def figura(self, chave, pagina, caixa):
        """Recorta uma região da página (foto/ilustração com rótulos vetoriais) e devolve o caminho da imagem."""
        if chave not in self._abertos:
            self._abertos[chave] = pymupdf.open(DOCS[chave][0])
        pg = self._abertos[chave][pagina - 1]
        r = pymupdf.Rect(caixa) & pg.rect
        zoom = min(3.0, 1100 / r.width)  # até ~1100 px de largura
        pix = pg.get_pixmap(matrix=pymupdf.Matrix(zoom, zoom), clip=r, alpha=False)
        im = Image.frombytes("RGB", (pix.width, pix.height), pix.samples)
        caminho = f"img/{chave}-p{pagina:02d}-{int(r.x0)}-{int(r.y0)}.jpg"
        self.arquivos[caminho] = (self._jpg(im, 78), "image/jpeg")
        return caminho

    @staticmethod
    def _jpg(im, q):
        b = io.BytesIO()
        im.save(b, "JPEG", quality=q, optimize=True, progressive=True)
        return b.getvalue()

    def ilustracao48(self):
        d = pymupdf.open(DOCS["doc259"][0])
        pg = d[18]  # p.19: modelo ilustrado
        pix = pg.get_pixmap(dpi=60, alpha=False)
        im = Image.frombytes("RGB", (pix.width, pix.height), pix.samples)
        w, h = im.size
        s = min(w, h)
        im = im.crop(((w - s) // 2, (h - s) // 2, (w + s) // 2, (h + s) // 2)).resize((48, 48), Image.LANCZOS)
        b = io.BytesIO()
        im.save(b, "PNG")
        return b.getvalue()


FIG = re.compile(r"^\{\{fig (\w+) (\d+) ([\d.,]+) \| (.+?) \| (.+?)( \| meia)?\}\}$", re.M)


def figuras(texto, midia):
    def sub(m):
        chave, pag, caixa, leg, cred, meia = m.groups()
        src = midia.figura(chave, int(pag), [float(x) for x in caixa.split(",")])
        cls = "fig meia" if meia else "fig"
        return (f'<figure class="{cls}" markdown="0"><a href="/{src}"><img loading="lazy" src="/{src}" '
                f'alt="{html.escape(leg)}"></a><figcaption>{html.escape(leg)} '
                f'<span class="cred">{html.escape(cred)} · {html.escape(DOCS[chave][2])}, p. {pag}</span>'
                f'</figcaption></figure>')
    # figuras "meia" consecutivas ficam lado a lado
    texto = FIG.sub(sub, texto)
    return re.sub(r'((?:<figure class="fig meia".*?</figure>\n?){2,})',
                  lambda m: '<div class="par-figs" markdown="0">' + m.group(1).replace("\n", "") + "</div>\n", texto)


# ---------------------------------------------------------------- páginas
class Pagina:
    def __init__(self, caminho, titulo, secao, corpo_html, front=True):
        self.caminho, self.titulo, self.secao, self.corpo, self.front = caminho, titulo, secao, corpo_html, front


def md_para_html(texto, midia):
    texto = figuras(texto, midia)
    return markdown.markdown(texto, extensions=MD_EXT, output_format="html5")


def gerar_faq(paginas_md):
    itens = []
    for cam in SUMARIO[2][1][1:]:
        meta, txt = paginas_md[cam]
        caps = []
        for m in re.finditer(r"^## (\d+)\. (.+?) \{#(q\d+)\}$", txt, flags=re.M):
            caps.append(f'<li><a href="/{cam}#{m.group(3)}"><b>{m.group(1)}.</b> {html.escape(m.group(2))}</a></li>')
        h1 = re.search(r"^# (.+)$", txt, flags=re.M).group(1)
        itens.append(f'<h2><a href="/{cam}">{html.escape(h1)}</a></h2><ol class="faq-lista" style="list-style:none;'
                     f'padding-left:0">{"".join(caps)}</ol>')
    corpo = ("<h1>Tanque de peixes – as 52 perguntas e respostas</h1>"
             "<p>Perguntas enviadas ao SAC da Embrapa (2014–2015) e respondidas no "
             '<a href="/fontes#doc259">Documentos 259 – Módulo 1: tanque de peixes</a> (Embrapa Meio-Norte, 2019). '
             "Clique numa pergunta para ir direto à resposta.</p>" + "".join(itens))
    return Pagina("faq", "Tanque de peixes – índice das 52 perguntas e respostas",
                  "Tanque de peixes – Perguntas e respostas", corpo)


def indice_portal():
    """Índice completo, em caixas por seção, para a página inicial (estilo portal da Wikipédia)."""
    caixas = []
    for secao, cams in SUMARIO:
        cams = [c for c in cams if c != "index"]
        if cams:
            itens = "".join(f'<li><a href="/{c}">{html.escape(TITULO_CURTO[c])}</a></li>' for c in cams)
            caixas.append(f'<div class="caixa"><h3>{html.escape(secao)}</h3><ul>{itens}</ul></div>')
    return f'<section id="indice" class="portal"><h2>Índice completo</h2>{"".join(caixas)}</section>'


def nesta_secao(atual):
    for secao, cams in SUMARIO:
        if atual in cams and len(cams) > 1:
            itens = "".join(f'<li><a{" class=\"atual\"" if c == atual else ""} href="/{c}">'
                            f'{html.escape(TITULO_CURTO[c])}</a></li>' for c in cams)
            return (f'<nav class="nesta-secao"><h4>Nesta seção: {html.escape(secao)}</h4><ul>{itens}</ul>'
                    f'<a class="todo-indice" href="/index#indice">Ver índice completo</a></nav>')
    return '<nav class="nesta-secao"><a class="todo-indice" href="/index#indice">Ver índice completo</a></nav>'


ORDEM = [c for _, cs in SUMARIO for c in cs]


def envolver(p: Pagina):
    i = ORDEM.index(p.caminho) if p.caminho in ORDEM else -1
    pag = ""
    if i >= 0:
        ant = f'<a href="/{ORDEM[i-1]}">← {html.escape(TITULO_CURTO[ORDEM[i-1]])}</a>' if i > 0 else "<span></span>"
        prox = (f'<a href="/{ORDEM[i+1]}">{html.escape(TITULO_CURTO[ORDEM[i+1]])} →</a>'
                if i + 1 < len(ORDEM) else "<span></span>")
        pag = f'<nav class="paginacao">{ant}{prox}</nav>'
    if p.caminho == "index":
        corpo = (p.corpo.replace("<!-- INDICE -->", indice_portal()) if "<!-- INDICE -->" in p.corpo
                 else p.corpo + indice_portal())
        migalha = ""
    else:
        corpo = p.corpo + nesta_secao(p.caminho)
        primeira = next((cs[0] for _, cs in SUMARIO if p.caminho in cs), None)
        secao = (f'<a href="/{primeira}">{html.escape(p.secao)}</a>' if primeira and primeira != p.caminho
                 else html.escape(p.secao))
        migalha = f'<div class="migalha"><a href="/index">Início</a> › {secao}</div>'
    doc = f"""<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(p.titulo)}</title>
<link rel="stylesheet" href="/{CSS}">
</head><body>
<header class="topo"><a class="marca" href="/index">Sisteminha Embrapa <span>· biblioteca offline</span></a>
<nav><a href="/index#indice">Índice</a><a href="/faq">Perguntas</a><a href="/glossario">Glossário</a><a href="/busca">Buscar</a></nav></header>
<main>{migalha}{corpo}{pag}</main>
<footer>Conteúdo baseado em publicações da Embrapa e parceiros – veja <a href="/fontes">Fontes</a>.
Coleção para uso pessoal e educativo.</footer>
</body></html>"""
    return relativizar(doc, p.caminho)


# ---------------------------------------------------------------- ZIM
class ItemZim(Item):
    def __init__(self, caminho, titulo, mime, dados, front=False):
        super().__init__()
        self.c, self.t, self.m, self.d, self.f = caminho, titulo, mime, dados, front

    def get_path(self): return self.c
    def get_title(self): return self.t
    def get_mimetype(self): return self.m
    def get_contentprovider(self): return StringProvider(self.d)
    def get_hints(self): return {Hint.FRONT_ARTICLE: self.f, Hint.COMPRESS: not self.m.startswith("image/")}


def main():
    html_dir = None
    if "--html-dir" in sys.argv:
        html_dir = Path(sys.argv[sys.argv.index("--html-dir") + 1])

    print("Processando PDFs e figuras…", file=sys.stderr)
    midia = Midia()
    midia.pdfs()

    paginas_md = {}
    for p in sorted(CONTEUDO.rglob("*.md")):
        cam = p.relative_to(CONTEUDO).with_suffix("").as_posix()
        paginas_md[cam] = ler_md(p)

    paginas = []
    for cam, (meta, txt) in paginas_md.items():
        corpo = md_para_html(txt, midia)
        chaves = [c.strip() for c in meta.get("fonte", "").split(",") if c.strip() in CREDITOS]
        corpo = com_credito(corpo, bloco_credito(chaves, meta.get("fonte_extra", "")))
        paginas.append(Pagina(cam, meta.get("titulo", cam), meta.get("secao", ""), corpo))
    paginas.append(gerar_faq(paginas_md))

    faltando = set(ORDEM) - {p.caminho for p in paginas}
    assert not faltando, faltando

    # índice para a busca em JavaScript (páginas de documentos originais ficam de fora: são grandes)
    indice = [{"t": p.titulo, "u": p.caminho, "s": p.secao, "x": texto_puro(p.corpo)}
              for p in paginas if p.caminho not in ("busca",)]
    js_indice = "window.INDICE=" + json.dumps(indice, ensure_ascii=False) + ";"

    estaticos = {
        CSS: ((ASSETS / "estilo.css").read_bytes(), "text/css"),
        "assets/busca.js": ((ASSETS / "busca.js").read_bytes(), "application/javascript"),
        "assets/calculadora.js": ((ASSETS / "calculadora.js").read_bytes(), "application/javascript"),
        "assets/indice-busca.js": (js_indice.encode(), "application/javascript"),
    }
    estaticos.update(midia.arquivos)

    htmls = {p.caminho: (p, envolver(p)) for p in paginas}

    # checagem de links internos quebrados
    existentes = set(htmls) | set(estaticos)
    quebrados = []
    for cam, (_, h) in htmls.items():
        base = cam.rsplit("/", 1)[0] + "/" if "/" in cam else ""
        for alvo in re.findall(r'(?:href|src)="([^"]*)"', h):
            if ":" in alvo or alvo.startswith("#"):
                continue
            alvo = re.split(r"[#?]", alvo)[0]
            partes = (base + alvo).split("/")
            pilha = []
            for x in partes:
                if x == "..":
                    pilha.pop()
                elif x and x != ".":
                    pilha.append(x)
            if "/".join(pilha) not in existentes:
                quebrados.append((cam, alvo))
    if quebrados:
        print("LINKS QUEBRADOS:", quebrados, file=sys.stderr)
        sys.exit(1)

    if html_dir:  # cópia em HTML puro para conferência no navegador
        for cam, (_, h) in htmls.items():
            f = html_dir / (cam + ".html")
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_text(re.sub(r'href="([^"#:]+?)(#[^"]*)?"',
                                lambda m: f'href="{m.group(1)}{"" if "." in m.group(1).rsplit("/",1)[-1] else ".html"}{m.group(2) or ""}"',
                                h), encoding="utf-8")
        for cam, (dados, _) in estaticos.items():
            f = html_dir / cam
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_bytes(dados)

    print(f"Gravando {SAIDA.name}…", file=sys.stderr)
    SAIDA.unlink(missing_ok=True)
    hoje = datetime.date.today().isoformat()
    with Creator(str(SAIDA)).config_indexing(True, "por") as z:
        z.set_mainpath("index")
        z.add_illustration(midia.ilustracao48(), 48)
        for k, v in {
            "Name": "sisteminha-embrapa_pt_all",
            "Title": "Sisteminha Embrapa",
            "Description": "Sisteminha Embrapa: produção integrada de peixes, ovos e hortaliças no quintal",
            "LongDescription": ("Biblioteca offline sobre o Sisteminha Embrapa-UFU-Fapemig (Sistema Integrado de "
                                "Produção de Alimentos): as 52 perguntas e respostas do tanque de peixes, biofiltro, "
                                "sedimentador, motobombas, galinhas, compostagem, horta, Sisteminha Comunidades, "
                                "calculadora de ração e os documentos originais da Embrapa."),
            "Language": "por",
            "Creator": "Embrapa",
            "Publisher": "Coleção pessoal",
            "Date": hoje,
            "Tags": "agricultura;piscicultura",
            "Source": "https://www.embrapa.br/sisteminha",
        }.items():
            z.add_metadata(k, v)
        for cam, (p, h) in htmls.items():
            z.add_item(ItemZim(cam, p.titulo, "text/html", h, front=cam not in ("busca",)))
        for cam, (dados, mime) in estaticos.items():
            z.add_item(ItemZim(cam, "", mime, dados))
        # endereços antigos (visualizador de páginas removido) -> página de fontes
        for k in DOCS:
            z.add_redirection(f"docs/{k}", f"Fonte: {DOCS[k][1]}", "fontes", {Hint.FRONT_ARTICLE: False})
    print(f"OK: {SAIDA} ({SAIDA.stat().st_size/1e6:.1f} MB, {len(htmls)} páginas, {len(estaticos)} arquivos)",
          file=sys.stderr)


if __name__ == "__main__":
    main()
