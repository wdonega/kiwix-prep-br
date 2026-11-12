# 🐟 Sisteminha Embrapa

Guia offline do **Sisteminha Embrapa**: um sistema integrado de produção de alimentos para a agricultura familiar (tanque de peixes com biofiltro, galinhas, codornas, porquinhos-da-índia, horta, compostagem…). Os PDFs da Embrapa foram reescritos como páginas navegáveis e pesquisáveis, com figuras recortadas dos originais e crédito da fonte no topo de cada página.

**Baixar:** [⬇️ sisteminha-embrapa_pt.zim](https://github.com/wdonega/kiwix-prep-br/releases/download/sisteminha-embrapa-latest/sisteminha-embrapa_pt.zim) (versão mais recente) · [versões anteriores](https://github.com/wdonega/kiwix-prep-br/releases?q=sisteminha-embrapa&expanded=true)

## Fontes

Todos os PDFs estão em [`fontes/`](fontes) e são citados página a página:

- GUILHERME, L. C. et al. *Sisteminha Embrapa-UFU-FAPEMIG – Módulo 1: tanque de peixes*. Embrapa Meio-Norte, 2019 (Documentos 259).
- RODRIGUES, L. A. et al. *Sisteminha Embrapa/UFU/Fapemig*. In: *Tecnologias para a Agricultura Familiar*, 4. ed., Embrapa, 2024 (Documentos 122).
- Embrapa Cocais. *Sisteminha Comunidades* (folder), 2024.
- Embrapa Meio-Norte. *Sistema integrado alternativo para produção de alimentos* (folders de 2012 e 2013).
- AVILA, V. S. de et al. *Sistema para produção de ovos com a poedeira Embrapa 051*. Embrapa Suínos e Aves, 2006 (Instrução Técnica 29).

## Estrutura

- `conteudo/*.md` – artigos em Markdown. Cabeçalho: `titulo`, `secao`, `fonte` (chaves dos PDFs), `fonte_extra`.
- Figuras: `{{fig CHAVE PÁGINA x0,y0,x1,y1 | legenda | crédito}}` (coordenadas em pontos do PDF; acrescente ` | meia` para meia largura).
- `assets/` – CSS, busca em JS e calculadora do tanque.
- `fontes/` – PDFs originais.
- `build.py` – gera `sisteminha-embrapa_pt_AAAA-MM.zim` (índice full-text em português).

## Gerar localmente

```bash
python3 -m venv venv && venv/bin/pip install -r ../requirements.txt
venv/bin/python build.py            # --html-dir /tmp/html para ver também em HTML solto
```
