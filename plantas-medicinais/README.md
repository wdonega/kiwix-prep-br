# 🌿 Plantas Medicinais

Biblioteca offline sobre **plantas medicinais e fitoterapia**, montada a partir de publicações oficiais da Anvisa, do Ministério da Saúde e da Embrapa:

- **85 plantas** do *Formulário de Fitoterápicos da Farmacopeia Brasileira* (3ª ed., 2026), com fórmula, preparo, dose, indicações e advertências, além de foto e índices por nome, indicação e forma de preparo.
- **Preparo e manipulação:** chás, tinturas, xaropes, uso externo, secagem e armazenamento.
- **Cultivo:** planejamento, solo, propagação, tratos culturais, colheita e espécies.
- **Uso seguro e SUS:** riscos, Farmácias Vivas, políticas e regulamentação.

> ⚕️ Material educativo. Não substitui a avaliação de um profissional de saúde.

**Baixar:** [releases `plantas-medicinais-*`](https://github.com/wdonega/kiwix-prep-br/releases?q=plantas-medicinais&expanded=true)

## Fontes

PDFs em [`fontes/`](fontes), citados página a página:

- ANVISA. *Formulário de Fitoterápicos da Farmacopeia Brasileira*, 3. ed., 2026. A obra autoriza reprodução, citada a fonte, para fins não comerciais.
- MINISTÉRIO DA SAÚDE. *Plantas medicinais e fitoterapia na Atenção Básica* (Cadernos de Atenção Básica 31), 2012.
- EMBRAPA RONDÔNIA. *Cultivo, uso e manipulação de plantas medicinais* (Documentos 91), 2004.
- EMBRAPA TABULEIROS COSTEIROS. *Orientações técnicas para o cultivo de plantas medicinais, aromáticas e condimentares* (Circular Técnica 70), 2015.
- EMBRAPA RORAIMA. *Cultivo de plantas medicinais em pequena escala* (Comunicado Técnico 6), 1999.
- EMBRAPA AMAZÔNIA ORIENTAL. *Plantas medicinais: uso e manipulação* (folder), 2003.

**Fotos:** Wikimedia Commons, todas com licença livre. Autor e licença de cada foto ficam em [`fotos/creditos.json`](fotos/creditos.json) e na página "Créditos das fotos" do ZIM.

## Estrutura

- `formulario.py` – extrai as 85 monografias do Formulário (`fontes/formulario-fitoterapicos-fb3.pdf`).
- `conteudo/` – páginas em Markdown: `preparo/`, `cultivo/`, `sus/`, `index.md`, `busca.md`.
  Cabeçalho: `titulo`, `secao`, `fonte` (chaves: form3, cab31, doc91, ct70, roraima, amazonia), `curto` opcional.
  Figuras: `{{fig CHAVE PÁGINA x0,y0,x1,y1 | legenda | crédito}}` (acrescente ` | meia` para meia largura).
  A ordem das páginas de cada seção segue a ordem dos links no `index.md` da seção.
- `fotos.py` – baixa as fotos do Commons para `fotos/`. Os arquivos já estão no repositório e só é preciso rodar de novo para trocar uma foto: ponha o nome do arquivo do Commons em `fotos/commons.json` e rode `python fotos.py --refazer <slug>`.
- `build.py` – gera `plantas-medicinais_pt_AAAA-MM.zim`.

## Gerar localmente

```bash
python3 -m venv venv && venv/bin/pip install -r ../requirements.txt
venv/bin/python build.py            # --html-dir DIR para gerar também HTML solto
```
