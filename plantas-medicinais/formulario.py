"""Extrai as monografias (FTxxx) do Formulário de Fitoterápicos da Farmacopeia Brasileira (3ª ed.)
para uma estrutura de dados: nome científico, nomes populares, fórmulas, preparo, advertências etc."""
import re
import unicodedata

import pymupdf

CAB = re.compile(r"^Formulário de Fitoterápicos da Farmacopeia Brasileira, 3ª edição\s*(?:–\s*(F[TF][\d.]+-\d+))?\s*$")
SECOES_FIXAS = ["NOMENCLATURA POPULAR", "ORIENTAÇÕES PARA O PREPARO", "EMBALAGEM E ARMAZENAMENTO",
                "ADVERTÊNCIAS", "INDICAÇÕES", "MODO DE USAR", "REFERÊNCIAS"]
SINONIMOS = {"ADVERTÊNCIA": "ADVERTÊNCIAS", "INDICAÇÃO": "INDICAÇÕES", "REFERÊNCIA": "REFERÊNCIAS",
             "ORIENTAÇÃO PARA O PREPARO": "ORIENTAÇÕES PARA O PREPARO", "MODOS DE USAR": "MODO DE USAR"}
QTD = re.compile(r"^(?:(?:uma?|duas?|equivalente|qsp)\b.*|q\.?\s?s\.?(?:p\.?)?|qs|[\d.,]+(?:\s*(?:a|–|-|ou)\s*[\d.,]+)?\s*(?:g|mg|mL|ml|L|%|µg|mcg|gotas?|unidades?|UI|kg)?\b.*)$", re.I)


SLUG_FIXO = {"FT020": "crataegus", "FT030": "eucalyptus-globulus-oleo", "FT033": "foeniculum-vulgare-amargo",
             "FT034": "foeniculum-vulgare-doce", "FT049": "mentha-piperita", "FT069": "salix"}


def slug(s):
    s = unicodedata.normalize("NFD", s).encode("ascii", "ignore").decode().lower()
    return re.sub(r"[^a-z0-9]+", "-", s).strip("-")


def eh_titulo(l):
    t = l.strip()
    if len(t) < 3 or t.startswith("Fórmula") or not re.search(r"[A-ZÀ-Ú]", t):
        return False
    letras = re.sub(r"[^A-Za-zÀ-ÿ]", "", t)
    return letras.isupper() and len(letras) >= 3


def linhas_por_codigo(pdf):
    """Devolve {codigo: [(pagina, linha), ...]} sem cabeçalhos de página."""
    d = pymupdf.open(pdf)
    blocos = {}
    for i, pg in enumerate(d, 1):
        linhas = pg.get_text().split("\n")
        cod = None
        for l in linhas[:3]:
            m = CAB.match(l.strip())
            if m:
                cod = m.group(1)
                break
        if not cod:
            continue
        for l in linhas:
            if CAB.match(l.strip()):
                continue
            blocos.setdefault(cod, []).append((i, l.rstrip()))
    return blocos


def paragrafos(linhas):
    """Junta linhas quebradas em parágrafos (linha em branco ou 'Fórmula N:' inicia novo)."""
    out, atual = [], []
    for l in linhas:
        t = l.strip()
        if not t:
            if atual:
                out.append(" ".join(atual))
                atual = []
            continue
        if atual and (re.match(r"^(Fórmulas? [\d, ae]+[:.]|Uso (oral|externo|tópico|retal|vaginal|inalatório)\b|Cuidados adicionais)", t)
                      or (atual[-1].endswith(".") and re.match(r"^(Para |Fórmula)", t))):
            out.append(" ".join(atual))
            atual = []
        atual.append(t)
    if atual:
        out.append(" ".join(atual))
    return [re.sub(r"\s+", " ", p).replace("- ", "-").strip() if False else re.sub(r"\s+", " ", p).strip() for p in out]


def formulas(linhas):
    """Lê blocos 'Fórmula N (refs)' + tabela Componentes/Quantidade."""
    res, cur, comp = [], None, []
    for l in linhas:
        t = l.strip()
        if not t:
            continue
        m = re.match(r"^Fórmula\s*(\d*)\s*(\(.*|)$", t)
        if m:
            cur = {"n": int(m.group(1) or len(res) + 1), "ref": m.group(2).strip(), "itens": []}
            res.append(cur)
            comp = []
            continue
        if cur is None or t in ("Componentes", "Quantidade", "Componentes Quantidade"):
            continue
        if not comp and cur["itens"] and t[:1].islower() and not QTD.match(t):  # continuação da quantidade
            c0, q0 = cur["itens"][-1]
            cur["itens"][-1] = (c0, q0 + " " + t)
            continue
        if comp and QTD.match(t) and not re.match(r"^\d+\s*%?\s*[a-zà-ú]", t, re.I) or (comp and re.match(r"^[\d.,]+\s", t) and QTD.match(t)):
            cur["itens"].append((" ".join(comp), t))
            comp = []
        else:
            if not cur["itens"] and not comp and t.startswith("(") and not cur["ref"]:
                cur["ref"] = t
                continue
            comp.append(t)
    if cur is not None and comp:
        cur["itens"].append((" ".join(comp), ""))
    return res


def monografias(pdf):
    out = []
    for cod, linhas in linhas_por_codigo(pdf).items():
        if not cod.startswith("FT"):
            continue
        paginas = sorted({p for p, _ in linhas})
        txt = [l for _, l in linhas]
        # nome científico: linhas antes de NOMENCLATURA POPULAR
        i_nom = next(i for i, l in enumerate(txt) if l.strip() == "NOMENCLATURA POPULAR")
        nome = " ".join(l.strip() for l in txt[:i_nom] if l.strip())
        sinonimia = ""
        if " SINONÍMIA " in nome:
            nome, sinonimia = nome.split(" SINONÍMIA ", 1)
        secoes, ordem, atual = {}, [], None
        for l in txt[i_nom:]:
            l = SINONIMOS.get(l.strip(), l)
            if eh_titulo(l) and (l.strip() in SECOES_FIXAS or atual in (None,) or
                                 ("ORIENTAÇÕES PARA O PREPARO" not in secoes)):
                atual = l.strip()
                ordem.append(atual)
                secoes.setdefault(atual, [])
                continue
            secoes[atual].append(l)
        formas = [s for s in ordem if s not in SECOES_FIXAS]
        populares = " ".join(secoes.get("NOMENCLATURA POPULAR", [])).strip().rstrip(".")
        m = {
            "codigo": cod, "paginas": paginas, "nome": nome, "sinonimia": sinonimia.strip(),
            "populares": [p.strip() for p in re.split(r",| e (?=[a-zà-ú])", populares) if p.strip()],
            "formas": [{"forma": f, "formulas": formulas(secoes[f])} for f in formas],
            "preparo": paragrafos(secoes.get("ORIENTAÇÕES PARA O PREPARO", [])),
            "embalagem": paragrafos(secoes.get("EMBALAGEM E ARMAZENAMENTO", [])),
            "advertencias": paragrafos(secoes.get("ADVERTÊNCIAS", [])),
            "indicacoes": paragrafos(secoes.get("INDICAÇÕES", [])),
            "modo": paragrafos(secoes.get("MODO DE USAR", [])),
            "referencias": [r for r in paragrafos(secoes.get("REFERÊNCIAS", [])) if r],
        }
        base = re.match(r"^([A-Z][a-z]+ [a-z-]+)", nome)
        parte = re.search(r"\((córtex|semente|raiz|planta inteira|folha|flor|fruto|casca|rizoma|óleo[^)]*)\)", nome)
        m["slug"] = SLUG_FIXO.get(cod[:5]) or slug((base.group(1) if base else nome) + (" " + parte.group(1) if parte else ""))
        out.append(m)
    return out


if __name__ == "__main__":
    import json
    import sys
    ms = monografias(sys.argv[1])
    json.dump(ms, open(sys.argv[2], "w"), ensure_ascii=False, indent=1)
    for m in ms:
        print(m["codigo"], m["slug"], "|", ", ".join(m["populares"])[:60], "|", [f["forma"] for f in m["formas"]],
              [len(f["formulas"]) for f in m["formas"]])
