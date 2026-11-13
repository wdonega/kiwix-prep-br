# 🎒 kiwix-prep-br

**Conhecimento útil que funciona sem internet, em português.**

Quando a rede cai, numa enchente, num apagão, no sítio ou numa viagem, o que você precisa saber tem que estar **no seu computador, no celular ou num pendrive**. Este repositório gera acervos offline no formato **ZIM**, para abrir no [Kiwix](https://kiwix.org/), com foco em **preparação, autonomia e saúde**. Também reúne em um só lugar os outros acervos em português que valem a pena baixar.

> Projeto independente, sem relação oficial com o Kiwix ou com a Wikimedia. Os ZIMs são feitos para abrir no Kiwix.

### 👀 Ver online antes de baixar

Os acervos estão no ar em **[prep.bigode.org](https://prep.bigode.org/)**, servidos pelo próprio Kiwix: dá para navegar e buscar do navegador, do jeito que fica depois de baixado.

---

## 📦 Acervos deste repositório

| | Acervo | O que tem | Baixar |
|---|---|---|---|
| 🐟 | [**Sisteminha Embrapa**](sisteminha-embrapa) | Produção integrada de alimentos para a agricultura familiar: tanque de peixes com biofiltro, aves, horta e compostagem. Reescrito a partir dos PDFs da Embrapa, com figuras, calculadora e busca. | **[⬇️ sisteminha-embrapa_pt.zim](https://github.com/wdonega/kiwix-prep-br/releases/download/sisteminha-embrapa-latest/sisteminha-embrapa_pt.zim)** (~16 MB)<br><sub>[sha256](https://github.com/wdonega/kiwix-prep-br/releases/download/sisteminha-embrapa-latest/sisteminha-embrapa_pt.zim.sha256) · [versões anteriores](https://github.com/wdonega/kiwix-prep-br/releases?q=sisteminha-embrapa&expanded=true)</sub> |
| 🌿 | [**Plantas Medicinais**](plantas-medicinais) | 85 plantas do Formulário de Fitoterápicos da Anvisa (fórmula, preparo, dose, advertências e foto), com seções de preparo caseiro, cultivo e uso seguro no SUS. | **[⬇️ plantas-medicinais_pt.zim](https://github.com/wdonega/kiwix-prep-br/releases/download/plantas-medicinais-latest/plantas-medicinais_pt.zim)** (~29 MB)<br><sub>[sha256](https://github.com/wdonega/kiwix-prep-br/releases/download/plantas-medicinais-latest/plantas-medicinais_pt.zim.sha256) · [versões anteriores](https://github.com/wdonega/kiwix-prep-br/releases?q=plantas-medicinais&expanded=true)</sub> |

O link de download é fixo e sempre aponta para a versão mais recente. Cada página do acervo cita a publicação e a página de origem, e os PDFs originais vão junto dentro do ZIM.

### Como as releases são geradas

Cada acervo mora numa pasta e tem um workflow próprio no GitHub Actions. Um **push na `main` que mexe numa pasta** gera de novo **só aquele ZIM** e publica uma release com tag `<pasta>-AAAA-MM-DD-rN`, com o `.zim` e o `.sha256` anexados. A release `<pasta>-latest` também é atualizada; é ela que mantém o link fixo da tabela acima. Mudanças em outras pastas não disparam o build. Também dá para rodar à mão em *Actions → Run workflow*.

Para adicionar um acervo novo:
1. Crie uma pasta com um `build.py` que gere um `*.zim` dentro dela.
2. Copie `.github/workflows/plantas-medicinais.yml` trocando o nome da pasta.

---

## 📚 Wikipédia em português (oficial do Kiwix)

Os arquivos são gerados pelo próprio Kiwix e ficam em [download.kiwix.org/zim/wikipedia](https://download.kiwix.org/zim/wikipedia/). Os nomes mudam a cada edição; os da tabela são os de **nov/2026**.

| Versão | O que é | Tamanho | Arquivo |
|---|---|---|---|
| **Completa com imagens** (`maxi`) | Todos os artigos, com fotos | ~19 GB | [`wikipedia_pt_all_maxi_2026-05.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_all_maxi_2026-05.zim) |
| **Completa sem imagens** (`nopic`) | Todos os artigos, só texto | ~6,6 GB | [`wikipedia_pt_all_nopic_2026-05.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_all_nopic_2026-05.zim) |
| **Resumida** (`mini`) | Só a introdução de cada artigo | ~1,6 GB | [`wikipedia_pt_all_mini_2026-05.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_all_mini_2026-05.zim) |
| **Top com imagens** | Os artigos mais lidos, com fotos | ~2,8 GB | [`wikipedia_pt_top_maxi_2026-10.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_top_maxi_2026-10.zim) |
| **Top sem imagens** | Os artigos mais lidos, só texto | ~1 GB | [`wikipedia_pt_top_nopic_2026-10.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_top_nopic_2026-10.zim) |

### 🩺 Wikipédia Medicina (WikiProject Medicine) em português

Só os artigos de saúde, ótimo para ter no celular.

| Versão | Tamanho | Arquivo |
|---|---|---|
| Com imagens (`maxi`) | ~281 MB | [`wikipedia_pt_medicine_maxi_2026-10.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_medicine_maxi_2026-10.zim) |
| Sem imagens (`nopic`) | ~116 MB | [`wikipedia_pt_medicine_nopic_2026-10.zim`](https://download.kiwix.org/zim/wikipedia/wikipedia_pt_medicine_nopic_2026-10.zim) |

### Outros projetos Wikimedia em português

| Acervo | Para quê | Tamanho |
|---|---|---|
| [Wikilivros](https://download.kiwix.org/zim/wikibooks/) (`wikibooks_pt_all_nopic`) | Livros didáticos e manuais | ~55 MB |
| [Wikcionário](https://download.kiwix.org/zim/wiktionary/) (`wiktionary_pt_all_nopic`) | Dicionário | ~353 MB |
| [Wikiversidade](https://download.kiwix.org/zim/wikiversity/) (`wikiversity_pt_all_nopic`) | Cursos e material de estudo | ~44 MB |
| [Wikivoyage](https://download.kiwix.org/zim/wikivoyage/) (`wikivoyage_pt_all_nopic`) | Guia de viagem | ~17 MB |
| [Wikisource](https://download.kiwix.org/zim/wikisource/) (`wikisource_pt_all_nopic`) | Textos em domínio público | ~1,5 GB |

Para ver tudo o que existe em português, use [library.kiwix.org](https://library.kiwix.org/#lang=por).

💡 **Qual baixar?** Num celular com pouco espaço, comece pela *Medicina nopic* e pela *Top nopic*. Num notebook ou HD externo, pegue a *Completa maxi* junto com a *Medicina maxi*. Para um Raspberry Pi servindo a família toda, junte tudo e rode o `kiwix-serve` (veja abaixo).

---

## 🇧🇷 Outros projetos brasileiros de preparação offline

Ainda são poucos. Estes estão no GitHub e são de outras pessoas, então não foram auditados aqui. Confira a origem do conteúdo antes de confiar.

- [**sdalberto/ia-da-sobrevivencia**](https://github.com/sdalberto/ia-da-sobrevivencia): assistente de sobrevivência 100% offline (Windows, Linux e macOS), com guias com fontes sobre água, fogo, comida e primeiros socorros, IA local e bibliotecas Kiwix.
- [**brenobernardes25-afk/biblioteca-offline**](https://github.com/brenobernardes25-afk/biblioteca-offline): biblioteca que funciona sem internet, com enciclopédias, manuais de saúde, agricultura e sobrevivência, mapas e IA local em português.

Conhece outro? Abra uma *issue* ou um PR.

### Fontes oficiais boas para virar ZIM

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
- Os acervos da Wikipédia são do Kiwix e da Wikimedia, sob a licença CC BY-SA.

> ⚕️ Os acervos de saúde são educativos e **não substituem um profissional de saúde**.
