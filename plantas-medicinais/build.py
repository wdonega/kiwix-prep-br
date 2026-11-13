#!/usr/bin/env python3
"""Gera plantas-medicinais_pt_AAAA-MM.zim.

Conteúdo: monografias do Formulário de Fitoterápicos (3ª ed., Anvisa) extraídas automaticamente
(formulario.py) + seções escritas em conteudo/**/*.md (cultivo, preparo, sus).

Uso:  python build.py [--html-dir DIR]   (requer: libzim, pymupdf, markdown, pillow)
"""
import datetime
import hashlib
import html
import io
import json
import re
import sys
import unicodedata
from pathlib import Path

import markdown
import pymupdf
from PIL import Image
from libzim.writer import Creator, Hint, Item, StringProvider

import formulario

AQUI = Path(__file__).resolve().parent
FONTES = AQUI / "fontes"
import os
CONTEUDO = Path(os.environ.get("CONTEUDO_DIR", AQUI / "conteudo"))
ASSETS = AQUI / "assets"
# nome com hash do conteúdo: quando o CSS muda, a URL muda e o navegador não usa a versão velha do cache
CSS = f"assets/estilo-{hashlib.sha1((ASSETS / 'estilo.css').read_bytes()).hexdigest()[:8]}.css"
SAIDA = Path(os.environ.get("SAIDA_ZIM", AQUI / f"plantas-medicinais_pt_{datetime.date.today():%Y-%m}.zim"))
E = html.escape

# chave -> (arquivo, rótulo curto p/ citações e figuras, referência completa, URL original)
DOCS = {
    "form3": ("formulario-fitoterapicos-fb3.pdf", "Formulário FB 3ª ed.",
              "BRASIL. Agência Nacional de Vigilância Sanitária. <i>Formulário de Fitoterápicos da Farmacopeia "
              "Brasileira</i>. 3. ed. Brasília: Anvisa, 2026. Aprovado pela RDC nº 1.024, de 14 de maio de 2026.",
              "https://bibliotecadigital.anvisa.gov.br/jspui/handle/anvisa/21045"),
    "cab31": ("cab31-plantas-medicinais-fitoterapia.pdf", "Caderno de Atenção Básica 31",
              "BRASIL. Ministério da Saúde. <i>Práticas integrativas e complementares: plantas medicinais e "
              "fitoterapia na Atenção Básica</i>. Brasília: Ministério da Saúde, 2012. 156 p. (Cadernos de Atenção "
              "Básica, n. 31). (Série A. Normas e Manuais Técnicos). ISBN 978-85-334-1912-4.",
              "https://www.fsp.usp.br/lcsoabcpaulista/wp-content/uploads/2019/12/praticas_integrativas_complementares_plantas_medicinais_cab31.pdf"),
    "doc91": ("embrapa-doc91-cultivo-uso-manipulacao.pdf", "Doc. 91",
              "RODRIGUES, V. G. S. <i>Cultivo, uso e manipulação de plantas medicinais</i>. Porto Velho: Embrapa "
              "Rondônia, mar. 2004. 25 p. (Documentos, 91). ISSN 0103-9865.",
              "https://www.infoteca.cnptia.embrapa.br/bitstream/doc/916031/1/doc91plantasmedicinais.pdf"),
    "ct70": ("embrapa-ct70-orientacoes-cultivo.pdf", "CT 70",
             "CARVALHO, L. M. de. <i>Orientações técnicas para o cultivo de plantas medicinais, aromáticas e "
             "condimentares</i>. Aracaju: Embrapa Tabuleiros Costeiros, out. 2015. (Circular Técnica, 70). "
             "ISSN 1678-1945.",
             "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/1028982/1/CT70.pdf"),
    "roraima": ("embrapa-roraima-cultivo-pequena-escala.pdf", "Embrapa Roraima",
                "LUZ, F. J. de F.; MIGLIORE, R. C. <i>Cultivo de plantas medicinais em pequena escala</i>. Embrapa "
                "Roraima, nov. 1999. 9 p. (Comunicado Técnico, 6).",
                "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/689271/1/cot00699medjoa.pdf"),
    "amazonia": ("embrapa-amazonia-uso-manipulacao.pdf", "Embrapa Amazônia Oriental",
                 "EMBRAPA AMAZÔNIA ORIENTAL. <i>Plantas medicinais: uso e manipulação</i>. Folder. Belém, 2003. Equipe "
                 "técnica: O. A. Lameira, E. C. P. de Oliveira, J. S. Paiva, L. B. Teixeira, V. L. C. Germano.",
                 "https://www.infoteca.cnptia.embrapa.br/infoteca/bitstream/doc/381106/1/fd570001.pdf"),
}
# referências refinadas pelos redatores de cada seção (sobrescrevem DOCS[k][2] se existirem)
REFS_JSON = AQUI / "referencias.json"
FOTOS = AQUI / "fotos"
CREDITOS_FOTOS = json.loads((FOTOS / "creditos.json").read_text()) if (FOTOS / "creditos.json").exists() else {}
FOTOS_IGNORAR = set(json.loads((FOTOS / "ignorar.json").read_text())) if (FOTOS / "ignorar.json").exists() else set()


def foto_de(slug):
    if slug in FOTOS_IGNORAR or slug not in CREDITOS_FOTOS or not (FOTOS / f"{slug}.jpg").exists():
        return None
    return CREDITOS_FOTOS[slug]


def credito_foto(c):
    lic = (f'<a href="{E(c["licenca_url"])}">{E(c["licenca"])}</a>' if c.get("licenca_url") else E(c["licenca"]))
    return (f'Foto: {E(c["autor"])} · {lic} · <a href="{E(c["pagina"])}">Wikimedia Commons</a> '
            '<small>(online)</small>')

SECOES_MD = [  # (diretório, nome da seção) – ordem do sumário
    ("preparo", "Preparo e manipulação"),
    ("cultivo", "Cultivo"),
    ("sus", "Uso seguro e SUS"),
]

CATEGORIAS = [  # (id, rótulo, regex sobre o texto das indicações)
    ("digestao", "Digestão, gases e azia", r"dispep|gastr|flatul|digest|azia|pirose|antiacid|coleret|plenitude|distensao|colicas? intestina|inapetencia"),
    ("diarreia", "Diarreia", r"diarreia"),
    ("respiratorio", "Tosse, gripe e resfriado", r"tosse|resfriado|gripe|vias aereas|expectorante|congest"),
    ("ansiedade-sono", "Ansiedade, nervosismo e sono", r"ansiedade|insonia|sono\b|sedativo|tensao nervosa|ansiolitico"),
    ("pele", "Pele, feridas e queimaduras", r"pele|cutane|ferim|cicatriz|queimad|dermat|couro cabeludo|escoria|picada"),
    ("boca-garganta", "Boca e garganta", r"oral|orofar|faringe|afta"),
    ("dor-inflamacao", "Dores, pancadas e inflamação", r"dor articular|contus|entorse|hematoma|equimose|anti-?inflamat|inflamatorios localizados|migranea"),
    ("urinario", "Rins e vias urinárias", r"urinar|prostat|retencao hidrica"),
    ("circulacao", "Circulação, pernas e hemorroidas", r"circulat|pernas|hemorroid|ateroscler"),
    ("mulher", "Saúde da mulher (cólica, TPM, climatério)", r"dismenorreia|menstrua|climaterio"),
    ("febre", "Febre", r"febre"),
    ("cansaco", "Cansaço e fraqueza", r"astenia|fadiga"),
    ("enjoo", "Enjoo", r"cinetose|enjoo|nausea"),
    ("figado", "Fígado e icterícia", r"ictericia"),
]

MD_EXT = ["tables", "attr_list", "def_list", "admonition", "md_in_html", "fenced_code", "sane_lists", "toc"]
LIMITE_PDF = 15_000_000


def sem_acento(s):
    return unicodedata.normalize("NFD", s).encode("ascii", "ignore").decode().lower()


def sem_citacoes(t):
    t = re.sub(r"\s*\((?:[^()]|\([^()]*\))*\d{4}[a-z]?(?:[^()]|\([^()]*\))*\)", "", t)
    return re.sub(r"\s+([.;,])", r"\1", re.sub(r"\s+", " ", t)).strip()


def capitalizar(forma):
    f = forma.lower()
    return f[:1].upper() + f[1:]


# ---------------------------------------------------------------- mídia
class Midia:
    def __init__(self):
        self.arquivos, self._abertos, self.compactado = {}, {}, set()

    @staticmethod
    def _jpg(im, q):
        b = io.BytesIO()
        im.save(b, "JPEG", quality=q, optimize=True, progressive=True)
        return b.getvalue()

    def pdfs(self):
        for chave, (arq, *_r) in DOCS.items():
            caminho = FONTES / arq
            dados = caminho.read_bytes()
            if len(dados) > LIMITE_PDF:  # recompacta PDFs enormes (imagens em JPEG)
                d = pymupdf.open(caminho)
                d.rewrite_images(dpi_threshold=160, dpi_target=150, quality=70)
                dados = d.tobytes(garbage=3, deflate=True)
                self.compactado.add(chave)
            self.arquivos[f"pdf/{arq}"] = (dados, "application/pdf")
            print(f"  pdf {chave}: {len(dados)/1e6:.1f} MB", file=sys.stderr)

    def fotos(self):
        for slug in CREDITOS_FOTOS:
            if not foto_de(slug):
                continue
            dados = (FOTOS / f"{slug}.jpg").read_bytes()
            self.arquivos[f"fotos/{slug}.jpg"] = (dados, "image/jpeg")
            im = Image.open(io.BytesIO(dados))
            im.thumbnail((260, 200))
            self.arquivos[f"fotos/t-{slug}.jpg"] = (self._jpg(im, 75), "image/jpeg")

    def figura(self, chave, pagina, caixa):
        if chave not in self._abertos:
            self._abertos[chave] = pymupdf.open(FONTES / DOCS[chave][0])
        pg = self._abertos[chave][pagina - 1]
        r = pymupdf.Rect(caixa) & pg.rect
        zoom = min(3.0, 1100 / max(r.width, 1))
        pix = pg.get_pixmap(matrix=pymupdf.Matrix(zoom, zoom), clip=r, alpha=False)
        im = Image.frombytes("RGB", (pix.width, pix.height), pix.samples)
        caminho = f"img/{chave}-p{pagina:03d}-{int(r.x0)}-{int(r.y0)}.jpg"
        self.arquivos[caminho] = (self._jpg(im, 78), "image/jpeg")
        return caminho

    @staticmethod
    def ilustracao48():
        # folha estilizada desenhada (sem depender de imagem de terceiros)
        im = Image.new("RGB", (192, 192), (46, 125, 50))
        from PIL import ImageDraw
        d = ImageDraw.Draw(im)
        d.ellipse((40, 30, 152, 170), fill=(200, 230, 201))
        d.line((96, 40, 96, 175), fill=(46, 125, 50), width=8)
        for y in (70, 100, 130):
            d.line((96, y + 15, 60, y), fill=(46, 125, 50), width=5)
            d.line((96, y + 15, 132, y), fill=(46, 125, 50), width=5)
        im = im.resize((48, 48), Image.LANCZOS)
        b = io.BytesIO()
        im.save(b, "PNG")
        return b.getvalue()


FIG = re.compile(r"^\{\{fig (\w+) (\d+) ([\d.,]+) \| (.+?) \| (.+?)( \| meia)?\}\}\s*$", re.M)


def figuras(texto, midia):
    def sub(m):
        chave, pag, caixa, leg, cred, meia = m.groups()
        src = midia.figura(chave, int(pag), [float(x) for x in caixa.split(",")])
        return (f'<figure class="fig{" meia" if meia else ""}" markdown="0"><a href="/{src}"><img loading="lazy" '
                f'src="/{src}" alt="{E(leg)}"></a><figcaption>{E(leg)} <span class="cred">{E(cred)} · '
                f'{E(DOCS[chave][1])}, p. {pag}</span></figcaption></figure>')
    texto = FIG.sub(sub, texto)
    return re.sub(r'((?:<figure class="fig meia".*?</figure>\n?){2,})',
                  lambda m: '<div class="par-figs" markdown="0">' + m.group(1).replace("\n", "") + "</div>\n", texto)


# ---------------------------------------------------------------- créditos
def ref_completa(k):
    if REFS_JSON.exists():
        r = json.loads(REFS_JSON.read_text()).get(k)
        if r:
            return r
    return DOCS[k][2]


def bloco_credito(chaves, extra=""):
    if not chaves:
        return ""
    itens = [f'<li>{ref_completa(k)} {extra} <a href="/pdf/{DOCS[k][0]}">PDF (cópia offline)</a> · '
             f'<a href="{DOCS[k][3]}">original na internet</a></li>' for k in chaves]
    return (f'<div class="credito"><b>Fonte</b><ul>{"".join(itens)}</ul>'
            f'<small>Texto reorganizado a partir do original, com citação de página. '
            f'Veja <a href="/fontes">todas as fontes</a>.</small></div>')


def com_credito(corpo, bloco):
    i = corpo.find("</h1>")
    return corpo[:i + 5] + bloco + corpo[i + 5:] if i >= 0 else bloco + corpo


AVISO = ('<div class="aviso-saude">⚕️ Informação de referência, baseada em publicações oficiais. '
         'Não substitui a avaliação de um profissional de saúde. Leia sempre as <b>advertências</b>; '
         'em caso de sintomas graves ou persistentes, procure o serviço de saúde.</div>')


# ---------------------------------------------------------------- páginas
class Pagina:
    def __init__(self, caminho, titulo, secao, corpo, curto=None, front=True):
        self.caminho, self.titulo, self.secao, self.corpo = caminho, titulo, secao, corpo
        self.curto, self.front = curto or titulo, front


def ler_md(p):
    s = p.read_text(encoding="utf-8")
    meta = {}
    if s.startswith("---\n"):
        cab, s = s[4:].split("\n---\n", 1)
        for linha in cab.splitlines():
            if ":" in linha:
                k, v = linha.split(":", 1)
                meta[k.strip()] = v.strip()
    return meta, s


def resumo(t, n=220):
    return t if len(t) <= n else t[:n].rsplit(" ", 1)[0] + "…"


def categorias_de(m):
    t = sem_acento(sem_citacoes(" ".join(m["indicacoes"])))
    return [c for c in CATEGORIAS if re.search(c[2], t)]


def pagina_planta(m, backlinks):
    pop = m["populares"]
    titulo = f'{pop[0]} ({m["nome"].split(" (")[0] if len(m["nome"]) < 60 else m["nome"][:57] + "…"})'
    p0, p1 = m["paginas"][0], m["paginas"][-1]
    foto = foto_de(m["slug"])
    fig = ""
    if foto:
        fig = (f'<figure class="foto-planta"><a href="/fotos/{m["slug"]}.jpg"><img src="/fotos/{m["slug"]}.jpg" '
               f'alt="{E(pop[0])} ({E(m["nome"])})"></a><figcaption>{credito_foto(foto)}</figcaption></figure>')
    h = [f'<h1>{E(pop[0])} <small class="cient"><i>{E(m["nome"])}</i></small></h1>',
         bloco_credito(["form3"], f"Monografia {m['codigo']}, p. {p0}–{p1}."), AVISO,
         fig + '<table class="ficha">',
         f'<tr><th>Nomes populares</th><td>{E(", ".join(pop))}</td></tr>',
         f'<tr><th>Nome científico</th><td><i>{E(m["nome"])}</i></td></tr>']
    if m["sinonimia"]:
        h.append(f'<tr><th>Sinonímia</th><td>{E(m["sinonimia"])}</td></tr>')
    cats = categorias_de(m)
    if cats:
        h.append('<tr><th>Para que é usada</th><td>' + " · ".join(
            f'<a href="/plantas/indicacoes#{c[0]}">{E(c[1])}</a>' for c in cats) + "</td></tr>")
    h.append(f'<tr><th>Formas</th><td>{E(", ".join(capitalizar(f["forma"]) for f in m["formas"]))}</td></tr>')
    h.append("</table>")
    if backlinks:
        h.append('<p class="veja">Veja também: ' + " · ".join(
            f'<a href="/{c}">{E(t)}</a>' for c, t in backlinks) + "</p>")

    def paras(lista, cls=""):
        return "".join(f'<p{cls}>{E(x)}</p>' for x in lista)

    h.append('<h2 id="indicacoes">Indicações</h2>' + paras(m["indicacoes"]))
    h.append('<h2 id="formulas">Fórmulas</h2>')
    for f in m["formas"]:
        h.append(f'<h3>{E(capitalizar(f["forma"]))}</h3><div class="formulas">')
        for fo in f["formulas"]:
            linhas = "".join(f"<tr><td>{E(c)}</td><td>{E(q)}</td></tr>" for c, q in fo["itens"])
            h.append(f'<table class="formula"><caption>Fórmula {fo["n"]} <small>{E(fo["ref"])}</small></caption>'
                     f"<tr><th>Componente</th><th>Quantidade</th></tr>{linhas}</table>")
        h.append("</div>")
    h.append('<h2 id="preparo">Como preparar</h2>' + paras(m["preparo"]))
    h.append('<p class="nota">Técnicas gerais (infusão, decocção, tintura, secagem) em '
             '<a href="/preparo/index">Preparo e manipulação</a>.</p>')
    h.append('<h2 id="modo">Modo de usar</h2>' + paras(m["modo"]))
    h.append('<h2 id="advertencias">Advertências</h2><div class="admonition danger">'
             + paras(m["advertencias"]) + "</div>")
    if m["embalagem"]:
        h.append('<h2 id="embalagem">Embalagem e armazenamento</h2>' + paras(m["embalagem"]))
    if m["referencias"]:
        h.append(f'<details class="refs"><summary>Referências citadas na monografia ({len(m["referencias"])})'
                 f'</summary><ul>{"".join(f"<li>{E(r)}</li>" for r in m["referencias"])}</ul></details>')
    return Pagina(f'plantas/{m["slug"]}', titulo, "Plantas (Formulário de Fitoterápicos)", "".join(h), pop[0])


def indices_plantas(ms):
    pags = []
    # A–Z por nome popular (cada nome popular entra)
    entradas = sorted(((p, m) for m in ms for p in m["populares"]), key=lambda x: sem_acento(x[0]))
    grupos = {}
    for p, m in entradas:
        grupos.setdefault(sem_acento(p)[0].upper(), []).append((p, m))
    letras = " ".join(f'<a href="#l-{L}">{L}</a>' for L in grupos)
    corpo = [f"<h1>Plantas de A a Z</h1>",
             f'<p>As {len(ms)} monografias do <b>Formulário de Fitoterápicos da Farmacopeia Brasileira</b> '
             f'(3ª ed., Anvisa, 2026), pelo nome popular. Veja também por <a href="/plantas/cientifico">nome '
             f'científico</a>, por <a href="/plantas/indicacoes">indicação</a> ou por '
             f'<a href="/plantas/formas">forma de preparo</a>.</p><p class="letras">{letras}</p>']
    for L, itens in grupos.items():
        corpo.append(f'<h2 id="l-{L}">{L}</h2><div class="grade-plantas">' + "".join(
            f'<a href="/plantas/{m["slug"]}">'
            + (f'<img loading="lazy" src="/fotos/t-{m["slug"]}.jpg" alt="">' if foto_de(m["slug"]) else
               '<span class="sem-foto">🌿</span>')
            + f'<b>{E(p[:1].upper() + p[1:])}</b><i>{E(m["nome"])}</i></a>' for p, m in itens) + "</div>")
    pags.append(Pagina("plantas/index", "Plantas medicinais de A a Z (nome popular)", "Plantas", "".join(corpo),
                       "A–Z (nome popular)"))
    # científico
    corpo = ["<h1>Plantas por nome científico</h1><ul class=\"lista-plantas\">"] + [
        f'<li><a href="/plantas/{m["slug"]}"><i>{E(m["nome"])}</i></a> – {E(", ".join(m["populares"]))} '
        f'<small>({m["codigo"]})</small></li>' for m in sorted(ms, key=lambda m: m["nome"])] + ["</ul>"]
    pags.append(Pagina("plantas/cientifico", "Plantas por nome científico", "Plantas", "".join(corpo),
                       "A–Z (nome científico)"))
    # indicações
    corpo = ["<h1>Plantas por indicação</h1>",
             '<p>Agrupamento feito a partir do texto de <b>Indicações</b> de cada monografia. Uma planta pode '
             'aparecer em mais de um grupo, e cada grupo pode valer só para uma das fórmulas – confira sempre a '
             'página da planta, as <b>advertências</b> e o modo de usar.</p>' + AVISO,
             '<p class="letras">' + " · ".join(f'<a href="#{c[0]}">{E(c[1])}</a>' for c in CATEGORIAS) + "</p>"]
    for cid, rot, _ in CATEGORIAS:
        sel = [m for m in ms if any(c[0] == cid for c in categorias_de(m))]
        corpo.append(f'<h2 id="{cid}">{E(rot)}</h2><ul class="lista-plantas">' + "".join(
            f'<li><a href="/plantas/{m["slug"]}">{E(m["populares"][0])}</a> <i>{E(m["nome"])}</i> – '
            f'<span class="ind">{E(resumo(sem_citacoes(" ".join(m["indicacoes"]))))}</span></li>'
            for m in sorted(sel, key=lambda m: sem_acento(m["populares"][0]))) + "</ul>")
    pags.append(Pagina("plantas/indicacoes", "Plantas medicinais por indicação (sintoma)", "Plantas",
                       "".join(corpo), "Por indicação"))
    # formas
    formas = {}
    for m in ms:
        for f in m["formas"]:
            formas.setdefault(capitalizar(f["forma"]), []).append(m)
    corpo = ["<h1>Plantas por forma de preparo</h1>",
             '<p>Formas farmacêuticas descritas no Formulário. Chás caseiros correspondem à '
             '<b>preparação extemporânea</b> (infusão ou decocção). Cápsulas, comprimidos, géis, cremes e pomadas '
             'são, em geral, preparados em farmácia de manipulação. Veja <a href="/preparo/index">Preparo e '
             'manipulação</a>.</p>']
    for f in sorted(formas, key=lambda f: (f != "Preparação extemporânea", f)):
        corpo.append(f'<h2>{E(f)} <small>({len(formas[f])})</small></h2><p>' + " · ".join(
            f'<a href="/plantas/{m["slug"]}">{E(m["populares"][0])}</a>'
            for m in sorted(formas[f], key=lambda m: sem_acento(m["populares"][0]))) + "</p>")
    pags.append(Pagina("plantas/formas", "Plantas por forma de preparo (chá, tintura, pomada…)", "Plantas",
                       "".join(corpo), "Por forma de preparo"))
    return pags


def pagina_fontes(midia):
    usos = {"form3": "Monografias das 85 plantas; definições e técnicas de manipulação.",
            "cab31": "Uso seguro, conceitos, políticas públicas, Farmácias Vivas, fitoterapia na Atenção Básica.",
            "doc91": "Cultivo, colheita, secagem e preparações caseiras.",
            "ct70": "Orientações técnicas de cultivo e qualidade da matéria-prima.",
            "roraima": "Cultivo em pequena escala e fichas de espécies.",
            "amazonia": "Uso e manipulação de plantas medicinais (preparos caseiros)."}
    linhas = "".join(
        f'<tr><td id="{k}">{ref_completa(k)}</td><td>{E(usos[k])}</td><td><a href="/pdf/{arq}">PDF offline</a>'
        f'{" <small>(imagens recompactadas)</small>" if k in midia.compactado else ""}<br><a href="{url}">'
        f'original</a> <small>(online)</small></td></tr>' for k, (arq, _r, _f, url) in DOCS.items())
    corpo = ("<h1>Fontes e documentos</h1><p>Todo o conteúdo desta coleção foi escrito a partir dos documentos "
             "abaixo, com citação de página em cada trecho (\"p.\" = página do PDF). As monografias das plantas "
             "foram extraídas automaticamente do Formulário e mantêm o texto original, com as referências "
             "bibliográficas de cada monografia.</p>"
             f"<table><tr><th>Documento</th><th>Usado em</th><th>Arquivo</th></tr>{linhas}</table>"
             "<h2>Direitos de uso</h2><ul>"
             "<li><b>Formulário de Fitoterápicos (Anvisa, 2026):</b> “É permitida a reprodução parcial ou total desta "
             "obra, desde que citada a fonte. É vedada a impressão, distribuição, reprodução desta obra para fins "
             "comerciais sem a prévia e expressa anuência da Anvisa.” <small>(p. 5)</small></li>"
             "<li><b>Ministério da Saúde e Embrapa:</b> publicações públicas, de acesso livre; os direitos pertencem "
             "aos autores e às instituições. Esta coleção é para uso pessoal e educativo, sem fins comerciais.</li></ul>"
             "<h2>Sobre esta coleção</h2><p>Montada em novembro de 2026 para leitura offline no Kiwix. Não substitui "
             "orientação de profissional de saúde.</p>")
    return Pagina("fontes", "Fontes e documentos", "Fontes", corpo, "Fontes")


def pagina_creditos_fotos(ms):
    linhas = []
    for m in sorted(ms, key=lambda m: formulario.slug(m["populares"][0])):
        c = foto_de(m["slug"])
        if c:
            linhas.append(f'<tr><td><a href="/plantas/{m["slug"]}">{E(m["populares"][0])}</a></td>'
                          f'<td><img loading="lazy" src="/fotos/t-{m["slug"]}.jpg" alt="" style="max-height:4rem"></td>'
                          f'<td>{E(c["arquivo"])}</td><td>{credito_foto(c)}</td></tr>')
    corpo = ("<h1>Créditos das fotos</h1><p>As fotos das plantas <b>não</b> vêm das publicações oficiais: são "
             "imagens de licença livre do <b>Wikimedia Commons</b>, escolhidas pela foto principal do artigo da "
             "Wikipédia sobre cada espécie (busca pelo nome científico). Servem para ilustrar; para identificar uma "
             "planta com segurança, siga as orientações de <a href=\"/cultivo/planejamento\">identificação "
             "correta</a> e procure um especialista.</p>"
             f"<p>{len(linhas)} fotos. Licenças CC BY / CC BY-SA exigem crédito ao autor e indicação da licença, "
             "como abaixo.</p><table><tr><th>Planta</th><th></th><th>Arquivo</th><th>Autor e licença</th></tr>"
             + "".join(linhas) + "</table>")
    return Pagina("creditos-fotos", "Créditos das fotos", "Fontes", corpo, "Créditos das fotos")


def ordem_secao(diretorio, md_paginas):
    """Ordem das páginas de uma seção = ordem dos links no index.md da seção."""
    cams = [c for c in md_paginas if c.startswith(diretorio + "/")]
    idx = f"{diretorio}/index"
    ordem = [idx] if idx in cams else []
    if idx in md_paginas:
        for alvo in re.findall(r"\]\(/(" + re.escape(diretorio) + r"/[\w-]+)", md_paginas[idx][1]):
            if alvo in cams and alvo not in ordem:
                ordem.append(alvo)
    return ordem + sorted(c for c in cams if c not in ordem)


# ---------------------------------------------------------------- HTML
def relativizar(h, caminho):
    pref = "../" * caminho.count("/")

    def sub(m):
        attr, alvo = m.group(1), m.group(2)[1:]
        if alvo == "" or alvo.startswith("#"):
            alvo = "index" + alvo
        return f'{attr}="{pref}{alvo}"'
    return re.sub(r'(href|src|action)="(/(?!/)[^"]*)"', sub, h)


def texto_puro(h):
    t = re.sub(r"<(script|style)[^>]*>.*?</\1>", " ", h, flags=re.S)
    return re.sub(r"\s+", " ", html.unescape(re.sub(r"<[^>]+>", " ", t))).strip()


def indice_portal(sumario):
    """Índice completo, em caixas por seção, para a página inicial (estilo portal da Wikipédia)."""
    caixas = []
    for secao, cams in sumario["secoes"]:
        cams = [(c, t) for c, t in cams if c != "index"]
        if cams:
            itens = "".join(f'<li><a href="/{c}">{E(t)}</a></li>' for c, t in cams)
            caixas.append(f'<div class="caixa"><h3>{E(secao)}</h3><ul>{itens}</ul></div>')
    return f'<section id="indice" class="portal"><h2>Índice completo</h2>{"".join(caixas)}</section>'


def envolver(p, sumario, ordem):
    i = ordem.index(p.caminho) if p.caminho in ordem else -1
    pag = ""
    if i >= 0:
        cur = {q.caminho: q for q in sumario["_todas"]}
        ant = (f'<a href="/{ordem[i-1]}">← {E(cur[ordem[i-1]].curto)}</a>' if i > 0 else "<span></span>")
        prox = (f'<a href="/{ordem[i+1]}">{E(cur[ordem[i+1]].curto)} →</a>' if i + 1 < len(ordem) else "<span></span>")
        pag = f'<nav class="paginacao">{ant}{prox}</nav>'
    secao_atual = next(((s, cs) for s, cs in sumario["secoes"] if any(c == p.caminho for c, _ in cs)), None)
    corpo = p.corpo
    if p.caminho == "index":
        corpo = corpo.replace("<!-- INDICE -->", indice_portal(sumario)) if "<!-- INDICE -->" in corpo \
            else corpo + indice_portal(sumario)
    elif secao_atual and len(secao_atual[1]) > 1:
        itens = "".join(f'<li><a{" class=\"atual\"" if c == p.caminho else ""} href="/{c}">{E(t)}</a></li>'
                        for c, t in secao_atual[1])
        corpo += (f'<nav class="nesta-secao"><h4>Nesta seção: {E(secao_atual[0])}</h4><ul>{itens}</ul>'
                  f'<a class="todo-indice" href="/index#indice">Ver índice completo</a></nav>')
    if p.caminho == "index":
        migalha = ""
    elif secao_atual and secao_atual[1][0][0] != p.caminho:
        migalha = (f'<div class="migalha"><a href="/index">Início</a> › '
                   f'<a href="/{secao_atual[1][0][0]}">{E(secao_atual[0])}</a></div>')
    elif p.caminho.startswith("plantas/"):
        migalha = '<div class="migalha"><a href="/index">Início</a> › <a href="/plantas/index">Plantas A–Z</a></div>'
    else:
        migalha = f'<div class="migalha"><a href="/index">Início</a> › {E(p.secao)}</div>'
    doc = f"""<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{E(p.titulo)}</title>
<link rel="stylesheet" href="/{CSS}">
</head><body>
<header class="topo"><a class="marca" href="/index">Plantas Medicinais <span>· biblioteca offline</span></a>
<nav><a href="/index#indice">Índice</a><a href="/plantas/index">Plantas A–Z</a><a href="/plantas/indicacoes">Por indicação</a><a href="/busca">Buscar</a></nav></header>
<main>{migalha}{corpo}{pag}</main>
<footer>Conteúdo baseado em publicações da Anvisa, do Ministério da Saúde e da Embrapa – veja <a href="/fontes">Fontes</a>.
Uso pessoal e educativo. Não substitui orientação profissional.</footer>
</body></html>"""
    return relativizar(doc, p.caminho)


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
    html_dir = Path(sys.argv[sys.argv.index("--html-dir") + 1]) if "--html-dir" in sys.argv else None
    midia = Midia()
    print("PDFs…", file=sys.stderr)
    midia.pdfs()
    midia.fotos()
    print("Monografias…", file=sys.stderr)
    ms = formulario.monografias(FONTES / DOCS["form3"][0])

    md = {p.relative_to(CONTEUDO).with_suffix("").as_posix(): ler_md(p) for p in sorted(CONTEUDO.rglob("*.md"))}

    # backlinks: páginas de seção que linkam cada planta
    back = {}
    for cam, (meta, txt) in md.items():
        if cam == "index":
            continue
        for slug in set(re.findall(r"\]\(/plantas/([\w-]+)", txt)):
            back.setdefault(slug, []).append((cam, meta.get("titulo", cam)))

    paginas = []
    for cam, (meta, txt) in md.items():
        corpo = markdown.markdown(figuras(txt, midia), extensions=MD_EXT, output_format="html5")
        chaves = [c.strip() for c in meta.get("fonte", "").split(",") if c.strip() in DOCS]
        corpo = com_credito(corpo, bloco_credito(chaves))
        paginas.append(Pagina(cam, meta.get("titulo", cam), meta.get("secao", ""), corpo, meta.get("curto")))
    plantas = [pagina_planta(m, back.get(m["slug"], [])) for m in ms]
    paginas += plantas + indices_plantas(ms) + [pagina_fontes(midia), pagina_creditos_fotos(ms)]

    porcam = {p.caminho: p for p in paginas}
    secoes = [("Início", ["index"]), ("Plantas", ["plantas/index", "plantas/indicacoes", "plantas/cientifico",
                                                  "plantas/formas"])]
    for d, nome in SECOES_MD:
        secoes.append((nome, ordem_secao(d, md)))
    secoes.append(("Mais", [c for c in ("glossario", "busca", "fontes", "creditos-fotos") if c in porcam]))
    for d, nome in SECOES_MD:  # título curto do index de seção
        if f"{d}/index" in porcam:
            porcam[f"{d}/index"].curto = "Visão geral"
    sumario = {"_todas": paginas,
               "secoes": [(s, [(c, porcam[c].curto) for c in cs if c in porcam]) for s, cs in secoes]}
    ordem = [c for _, cs in secoes for c in cs if c in porcam]
    ordem_plantas = [p.caminho for p in sorted(plantas, key=lambda p: sem_acento(p.curto))]

    def envolver_p(p):
        o = ordem_plantas if p.caminho in ordem_plantas else ordem
        return envolver(p, sumario, o)

    indice = [{"t": p.titulo, "u": p.caminho, "s": p.secao, "x": texto_puro(p.corpo)[:6000]}
              for p in paginas if p.caminho != "busca"]
    estaticos = {
        CSS: ((ASSETS / "estilo.css").read_bytes(), "text/css"),
        "assets/busca.js": ((ASSETS / "busca.js").read_bytes(), "application/javascript"),
        "assets/indice-busca.js": (("window.INDICE=" + json.dumps(indice, ensure_ascii=False) + ";").encode(),
                                   "application/javascript"),
    }
    estaticos.update(midia.arquivos)
    htmls = {p.caminho: (p, envolver_p(p)) for p in paginas}

    existentes = set(htmls) | set(estaticos)
    quebrados = []
    for cam, (_, h) in htmls.items():
        base = cam.rsplit("/", 1)[0] + "/" if "/" in cam else ""
        for alvo in re.findall(r'(?:href|src)="([^"]*)"', h):
            if ":" in alvo or alvo.startswith("#"):
                continue
            pilha = []
            for x in (base + re.split(r"[#?]", alvo)[0]).split("/"):
                if x == "..":
                    pilha.pop()
                elif x and x != ".":
                    pilha.append(x)
            if "/".join(pilha) not in existentes:
                quebrados.append((cam, alvo))
    if quebrados:
        print("LINKS QUEBRADOS:", *sorted(set(quebrados)), sep="\n  ", file=sys.stderr)
        sys.exit(1)

    if html_dir:
        for cam, (_, h) in htmls.items():
            f = html_dir / (cam + ".html")
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_text(re.sub(r'href="([^"#:]+?)(#[^"]*)?"', lambda m: f'href="{m.group(1)}'
                                f'{"" if "." in m.group(1).rsplit("/", 1)[-1] else ".html"}{m.group(2) or ""}"', h))
        for cam, (dados, _) in estaticos.items():
            f = html_dir / cam
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_bytes(dados)

    print(f"Gravando {SAIDA.name}…", file=sys.stderr)
    SAIDA.unlink(missing_ok=True)
    with Creator(str(SAIDA)).config_indexing(True, "por") as z:
        z.set_mainpath("index")
        z.add_illustration(Midia.ilustracao48(), 48)
        for k, v in {
            "Name": "plantas-medicinais_pt_all",
            "Title": "Plantas Medicinais",
            "Description": "Plantas medicinais: 85 monografias da Anvisa, preparo, cultivo e uso seguro",
            "LongDescription": ("Biblioteca offline sobre plantas medicinais e fitoterapia: as 85 monografias do "
                                "Formulário de Fitoterápicos da Farmacopeia Brasileira (3ª ed., Anvisa, 2026), "
                                "preparo e manipulação, cultivo (Embrapa) e uso seguro e fitoterapia no SUS "
                                "(Ministério da Saúde). Índices por nome popular, científico e indicação."),
            "Language": "por", "Creator": "Anvisa; Ministério da Saúde; Embrapa", "Publisher": "Coleção pessoal",
            "Date": datetime.date.today().isoformat(),
            "Tags": "saúde;fitoterapia",
            "Source": "https://bibliotecadigital.anvisa.gov.br/jspui/handle/anvisa/21045",
        }.items():
            z.add_metadata(k, v)
        for cam, (p, h) in htmls.items():
            z.add_item(ItemZim(cam, p.titulo, "text/html", h, front=cam != "busca"))
        for cam, (dados, mime) in estaticos.items():
            z.add_item(ItemZim(cam, "", mime, dados))
    print(f"OK: {SAIDA} ({SAIDA.stat().st_size/1e6:.1f} MB, {len(htmls)} páginas, {len(estaticos)} arquivos)",
          file=sys.stderr)


if __name__ == "__main__":
    main()
