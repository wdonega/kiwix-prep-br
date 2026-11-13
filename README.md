# 🎒 kiwix-prep-br

**Conhecimento útil que funciona sem internet, em português.**

Quando a rede cai, numa enchente, num apagão, no sítio ou numa viagem, o que você precisa saber tem que estar **no seu computador, no celular ou num pendrive**. Este repositório gera acervos offline no formato **ZIM**, para abrir no [Kiwix](https://kiwix.org/), com foco em **preparação, autonomia e saúde**. Também reúne num só script os outros acervos em português que valem a pena baixar.

> Projeto independente, sem relação oficial com o Kiwix ou com a Wikimedia. Os ZIMs são feitos para abrir no Kiwix.

### 👀 Confira o conteúdo antes de baixar

Todo o acervo deste README está no ar em **[prep.bigode.org](https://prep.bigode.org/#lang=por)**, servidos pelo próprio Kiwix: dá para navegar e buscar pelo navegador e ver exatamente o que vem no ZIM antes de baixar.

---

## 📦 Acervos deste repositório

| | Acervo | O que tem | Conferir | Baixar |
|---|---|---|---|---|
| 🐟 | [**Sisteminha Embrapa**](sisteminha-embrapa) | Produção integrada de alimentos para a agricultura familiar: tanque de peixes com biofiltro, aves, horta e compostagem. Reescrito a partir dos PDFs da Embrapa, com figuras, calculadora e busca. | **[👀 ver online](https://prep.bigode.org/content/sisteminha-embrapa_pt/index)** | **[⬇️ sisteminha-embrapa_pt.zim](https://github.com/wdonega/kiwix-prep-br/releases/download/sisteminha-embrapa-latest/sisteminha-embrapa_pt.zim)** (~16 MB)<br><sub>[sha256](https://github.com/wdonega/kiwix-prep-br/releases/download/sisteminha-embrapa-latest/sisteminha-embrapa_pt.zim.sha256) · [versões anteriores](https://github.com/wdonega/kiwix-prep-br/releases?q=sisteminha-embrapa&expanded=true)</sub> |
| 🌿 | [**Plantas Medicinais**](plantas-medicinais) | 85 plantas do Formulário de Fitoterápicos da Anvisa (fórmula, preparo, dose, advertências e foto), com seções de preparo caseiro, cultivo e uso seguro no SUS. | **[👀 ver online](https://prep.bigode.org/content/plantas-medicinais_pt/index)** | **[⬇️ plantas-medicinais_pt.zim](https://github.com/wdonega/kiwix-prep-br/releases/download/plantas-medicinais-latest/plantas-medicinais_pt.zim)** (~29 MB)<br><sub>[sha256](https://github.com/wdonega/kiwix-prep-br/releases/download/plantas-medicinais-latest/plantas-medicinais_pt.zim.sha256) · [versões anteriores](https://github.com/wdonega/kiwix-prep-br/releases?q=plantas-medicinais&expanded=true)</sub> |

O link de download é fixo e sempre aponta para a versão mais recente. Cada página do acervo cita a publicação e a página de origem, e os PDFs originais vão junto dentro do ZIM.

### Como as releases são geradas

Cada acervo mora numa pasta e tem um workflow próprio no GitHub Actions. Um **push na `main` que mexe numa pasta** gera de novo **só aquele ZIM** e publica uma release com tag `<pasta>-AAAA-MM-DD-rN`, com o `.zim` e o `.sha256` anexados. A release `<pasta>-latest` também é atualizada; é ela que mantém o link fixo da tabela acima. Mudanças em outras pastas não disparam o build. Também dá para rodar à mão em *Actions → Run workflow*.

Para adicionar um acervo novo:
1. Crie uma pasta com um `build.py` que gere um `*.zim` dentro dela.
2. Copie `.github/workflows/plantas-medicinais.yml` trocando o nome da pasta.

---

## 📚 Outros acervos em português

ZIMs oficiais do Kiwix, escolhidos para quem quer ter o essencial offline em português. Junto com os dois acervos acima, somam **~10 GB**.

| | Acervo | O que tem | Tamanho | Conferir |
|---|---|---|---|---|
| **Enciclopédias** | Wikipédia: os 50 mil melhores artigos | Artigos completos, com imagens | 3,0 GB | [👀 ver online](https://prep.bigode.org/content/wikipedia_pt_top_maxi_2026-10) |
|  | Wikipédia Medicina | Só os artigos de saúde, com imagens | 294 MB | [👀 ver online](https://prep.bigode.org/content/wikipedia_pt_medicine_maxi_2026-10) |
|  | Vikidia | Enciclopédia para crianças e adolescentes | 19 MB | [👀 ver online](https://prep.bigode.org/content/vikidia_pt_all_maxi_2026-09) |
| **Consertos** | iFixit | Guias de conserto passo a passo, com fotos | 3,6 GB | [👀 ver online](https://prep.bigode.org/content/ifixit_pt_all_2026-03) |
| **Estudo** | Wikilivros | Livros didáticos e manuais | 169 MB | [👀 ver online](https://prep.bigode.org/content/wikibooks_pt_all_maxi_2026-10) |
|  | Wikiversidade | Cursos e material de estudo | 101 MB | [👀 ver online](https://prep.bigode.org/content/wikiversity_pt_all_maxi_2026-10) |
|  | PhET | Simulações interativas de ciências e matemática | 81 MB | [👀 ver online](https://prep.bigode.org/content/phet_pt_all_2026-08) |
| **Livros e textos** | Wikisource | Textos em domínio público | 1,7 GB | [👀 ver online](https://prep.bigode.org/content/wikisource_pt_all_maxi_2026-10) |
|  | Projeto Gutenberg | Livros clássicos em português | 523 MB | [👀 ver online](https://prep.bigode.org/content/gutenberg_pt_all_2025-10) |
|  | Wikiquote | Citações | 90 MB | [👀 ver online](https://prep.bigode.org/content/wikiquote_pt_all_maxi_2026-10) |
| **Língua e viagem** | Wikcionário | Dicionário (sem imagens) | 370 MB | [👀 ver online](https://prep.bigode.org/content/wiktionary_pt_all_nopic_2026-09) |
|  | Portuguese Language Q&A | Perguntas e respostas sobre a língua portuguesa (a maioria em inglês) | 21 MB | [👀 ver online](https://prep.bigode.org/content/portuguese.stackexchange.com_mul_all_2026-10) |
|  | Wikivoyage | Guia de viagem | 57 MB | [👀 ver online](https://prep.bigode.org/content/wikivoyage_pt_all_maxi_2026-09) |

### ⬇️ Baixar tudo de uma vez

O script [`baixar-acervo-pt.sh`](baixar-acervo-pt.sh) baixa os 15 ZIMs, os dois deste repositório e os 13 acima, direto do GitHub e do [download.kiwix.org](https://download.kiwix.org/zim/):

```bash
cd /pasta/dos/zims
curl -fLO https://raw.githubusercontent.com/wdonega/kiwix-prep-br/main/baixar-acervo-pt.sh
sh baixar-acervo-pt.sh
```

- Se a conexão cair, rode de novo: o download continua de onde parou.
- Para pular algum, comente a linha com `#`. Os tamanhos estão ao lado de cada uma.
- Os nomes dos ZIMs do Kiwix têm a data da edição (são os de **nov/2026**). Quando sair uma edição nova e a antiga for apagada, o link dá erro 404: troque pela data mais recente em `download.kiwix.org/zim/<pasta>/`.
- Quer a Wikipédia **inteira**? Troque a linha da Top por [`wikipedia_pt_all_maxi_2026-05.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_all_maxi_2026-05.zim) (~20 GB, todos os artigos com imagens) ou [`wikipedia_pt_all_nopic_2026-05.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_all_nopic_2026-05.zim) (~6,6 GB, sem imagens).

---

## 🌱 Próximos acervos

Ideias para os próximos acervos, todas de órgãos públicos:

- **Embrapa** ([Infoteca-e](https://www.infoteca.cnptia.embrapa.br/)): cisternas, conservação de alimentos, pequenos animais, horta e biodigestor.
- **Defesa Civil**: cartilhas de enchente, deslizamento, seca e kit de emergência.
- **Ministério da Saúde** ([BVS](https://bvsms.saude.gov.br/)): primeiros socorros, tratamento de água e doenças de veiculação hídrica.
- **ASA Brasil**: cisterna de placas e convivência com o semiárido.

---

## 🖥️ Como usar os arquivos `.zim`

- **Celular:** app *Kiwix* (Android ou iOS). Abra o `.zim` baixado.
- **Computador:** [Kiwix Desktop](https://kiwix.org/pt-br/applications/).
- **Servidor na rede de casa** (Raspberry Pi, NAS, etc.):

```bash
docker run -d --name kiwix -v /caminho/dos/zims:/data -p 8080:8080 \
  ghcr.io/kiwix/kiwix-serve '*.zim'
# abra http://<ip-da-maquina>:8080 em qualquer aparelho da rede
```

## 🔧 Gerar localmente

```bash
python3 -m venv venv && venv/bin/pip install -r requirements.txt
cd plantas-medicinais && ../venv/bin/python build.py   # gera plantas-medicinais_pt_AAAA-MM.zim
```

## ⚖️ Licenças e créditos

- O **código** (`build.py`, scripts, CSS e JS) pode ser usado livremente.
- O **conteúdo** vem de publicações públicas da Embrapa, da Anvisa e do Ministério da Saúde, reorganizadas para uso **pessoal e educativo, sem fins comerciais**. Toda página cita a fonte. Os créditos completos estão no README de cada acervo e dentro de cada ZIM.
- As **fotos** do acervo de plantas vêm do Wikimedia Commons, com licenças livres, e o autor é creditado em cada imagem.
- Os outros acervos são gerados pelo Kiwix e seguem a licença de cada projeto de origem (os da Wikimedia, por exemplo, são CC BY-SA).

> ⚕️ Os acervos de saúde são educativos e **não substituem um profissional de saúde**.
