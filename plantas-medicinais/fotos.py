#!/usr/bin/env python3
"""Baixa uma foto (licença livre, Wikimedia Commons) para cada planta do Formulário.

Procura o artigo da Wikipédia pelo nome científico (en, depois pt), pega a imagem principal,
confere a licença no Commons e salva em fotos/<slug>.jpg + fotos/creditos.json (cache: só baixa o que falta).

Uso: python fotos.py [--refazer slug ...]
"""
import io
import json
import re
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path

from PIL import Image

import formulario

AQUI = Path(__file__).resolve().parent
FOTOS = AQUI / "fotos"
CRED = FOTOS / "creditos.json"
UA = {"User-Agent": "colecao-offline-plantas-medicinais/1.0 (uso pessoal; kiwix)"}
LICENCAS_OK = re.compile(r"^(cc0|cc[- ]by(-sa)?[- ]?[\d.]*|public domain|pd\b|attribution|gfdl)", re.I)

# slugs cuja foto vem direto da busca no Commons; lista vazia = listar candidatos; nomes = arquivos escolhidos
COMMONS_DIRETO = json.loads((Path(__file__).resolve().parent / "fotos" / "commons.json").read_text()) \
    if (Path(__file__).resolve().parent / "fotos" / "commons.json").exists() else {}

# nomes de busca quando o nome da monografia não é bom para procurar
BUSCA = {
    "salix": ["Salix purpurea", "Salix alba"],
    "crataegus": ["Crataegus monogyna"],
    "eucalyptus-globulus-oleo": ["Eucalyptus globulus"],
    "foeniculum-vulgare-amargo": ["Foeniculum vulgare"],
    "foeniculum-vulgare-doce": ["Foeniculum vulgare"],
    "mentha-piperita": ["Mentha × piperita", "Peppermint"],
    "monteverdia-ilicifolia": ["Monteverdia ilicifolia", "Maytenus ilicifolia"],
    "aesculus-hippocastanum-cortex": ["Aesculus hippocastanum"],
    "aesculus-hippocastanum-semente": ["Aesculus hippocastanum"],
    "echinacea-purpurea-planta-inteira": ["Echinacea purpurea"],
    "echinacea-purpurea-raiz": ["Echinacea purpurea"],
    "hamamelis-virginiana-cortex": ["Hamamelis virginiana"],
    "hamamelis-virginiana-folha": ["Hamamelis virginiana"],
    "libidibia-ferrea": ["Libidibia ferrea", "Caesalpinia ferrea"],
    "lippia-sidoides": ["Lippia sidoides", "Lippia origanoides"],
}


def get(url):
    time.sleep(1.2)
    for i in range(6):
        try:
            with urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=60) as r:
                return r.read()
        except Exception as e:  # rede instável / 429
            if i == 5:
                raise
            time.sleep(15 * (i + 1))


def api(host, **params):
    params.update(format="json", formatversion="2")
    return json.loads(get(f"https://{host}/w/api.php?" + urllib.parse.urlencode(params)))


def imagem_do_artigo(nome):
    for lang in ("en", "pt"):
        r = api(f"{lang}.wikipedia.org", action="query", titles=nome, redirects=1, prop="pageimages",
                piprop="name", pilicense="free")
        for p in r.get("query", {}).get("pages", []):
            if p.get("pageimage"):
                return p["pageimage"], f"https://{lang}.wikipedia.org/wiki/{urllib.parse.quote(p['title'])}"
    return None, None


def busca_commons(nome, n=8):
    r = api("commons.wikimedia.org", action="query", list="search", srnamespace=6, srlimit=n,
            srsearch=f'"{nome}" filetype:bitmap')
    return [x["title"].split(":", 1)[1] for x in r.get("query", {}).get("search", [])]


def info_commons(arquivo):
    r = api("commons.wikimedia.org", action="query", titles=f"File:{arquivo}", prop="imageinfo",
            iiprop="url|extmetadata|mime", iiurlwidth=900)
    pg = r["query"]["pages"][0]
    if "imageinfo" not in pg:
        return None
    ii = pg["imageinfo"][0]
    md = ii.get("extmetadata", {})
    limpa = lambda k: re.sub(r"<[^>]+>", "", md.get(k, {}).get("value", "")).strip()
    return {"arquivo": arquivo, "thumb": ii.get("thumburl") or ii["url"], "pagina": ii["descriptionurl"],
            "autor": re.sub(r"\s+", " ", limpa("Artist"))[:120] or "autor indicado na página do arquivo",
            "licenca": limpa("LicenseShortName"), "licenca_url": limpa("LicenseUrl"), "mime": ii.get("mime", "")}


def main():
    FOTOS.mkdir(exist_ok=True)
    cred = json.loads(CRED.read_text()) if CRED.exists() else {}
    refazer = set(sys.argv[sys.argv.index("--refazer") + 1:]) if "--refazer" in sys.argv else set()
    ms = formulario.monografias(AQUI / "fontes" / "formulario-fitoterapicos-fb3.pdf")
    for m in ms:
        s = m["slug"]
        if s in cred and s not in refazer and (FOTOS / f"{s}.jpg").exists():
            continue
        nomes = BUSCA.get(s) or [" ".join(m["nome"].split()[:2])]
        achou = None
        tentativas = []
        for nome in nomes:
            if s in COMMONS_DIRETO:
                tentativas += [(a, f"https://commons.wikimedia.org/w/index.php?search={urllib.parse.quote(nome)}", nome)
                               for a in busca_commons(nome)]
            else:
                arq, artigo = imagem_do_artigo(nome)
                if arq:
                    tentativas.append((arq, artigo, nome))
        for arq, artigo, nome in tentativas:
            if COMMONS_DIRETO.get(s) and arq not in COMMONS_DIRETO[s]:
                continue
            inf = info_commons(arq)
            if inf and LICENCAS_OK.match(inf["licenca"]) and inf["mime"] in ("image/jpeg", "image/png", "image/webp"):
                inf["artigo"], inf["busca"] = artigo, nome
                achou = inf
                break
            print(f"  {s}: rejeitada {arq} ({inf and inf['licenca']})", file=sys.stderr)
        if not achou:
            print(f"SEM FOTO: {s} ({nomes})", file=sys.stderr)
            continue
        im = Image.open(io.BytesIO(get(achou["thumb"]))).convert("RGB")
        im.thumbnail((900, 900))
        im.save(FOTOS / f"{s}.jpg", "JPEG", quality=80, optimize=True, progressive=True)
        cred[s] = {k: v for k, v in achou.items() if k != "thumb"}
        CRED.write_text(json.dumps(cred, ensure_ascii=False, indent=1))
        print(f"ok {s}: {achou['arquivo']} [{achou['licenca']}] {achou['autor'][:40]}")
        time.sleep(2)


if __name__ == "__main__":
    main()
