#!/bin/sh
# Baixa o acervo em português do prep-wiki-br (o mesmo que está em https://prep.bigode.org).
# Uso: rode dentro da pasta onde ficam os ZIMs, por exemplo:
#   cd /kiwix-data && sh baixar-acervo-pt.sh
# Se a conexão cair, rode de novo: o download continua de onde parou.
# Comente com # as linhas do que não quiser. Total: ~10 GB.
set -e
K=https://download.kiwix.org/zim
G=https://github.com/wdonega/prep-wiki-br/releases/download

baixar() { curl -fL --retry 5 --retry-delay 10 -C - -O "$1"; }

# ---- acervos deste repositório ----
baixar $G/sisteminha-embrapa-latest/sisteminha-embrapa_pt.zim          #  16 MB
baixar $G/plantas-medicinais-latest/plantas-medicinais_pt.zim          #  29 MB

# ---- enciclopédias ----
baixar $K/wikipedia/wikipedia_pt_top_maxi_2026-10.zim                  # 3,0 GB
baixar $K/wikipedia/wikipedia_pt_medicine_maxi_2026-10.zim             # 294 MB
baixar $K/vikidia/vikidia_pt_all_maxi_2026-09.zim                      #  19 MB

# ---- consertos ----
baixar $K/ifixit/ifixit_pt_all_2026-03.zim                             # 3,6 GB

# ---- estudo ----
baixar $K/wikibooks/wikibooks_pt_all_maxi_2026-10.zim                  # 169 MB
baixar $K/wikiversity/wikiversity_pt_all_maxi_2026-10.zim              # 101 MB
baixar $K/phet/phet_pt_all_2026-08.zim                                 #  81 MB

# ---- livros e textos ----
baixar $K/wikisource/wikisource_pt_all_maxi_2026-10.zim                # 1,7 GB
baixar $K/gutenberg/gutenberg_pt_all_2025-10.zim                       # 523 MB
baixar $K/wikiquote/wikiquote_pt_all_maxi_2026-10.zim                  #  90 MB

# ---- língua e viagem ----
baixar $K/wiktionary/wiktionary_pt_all_nopic_2026-09.zim               # 370 MB
baixar $K/stack_exchange/portuguese.stackexchange.com_mul_all_2026-10.zim  # 21 MB
baixar $K/wikivoyage/wikivoyage_pt_all_maxi_2026-09.zim                #  57 MB

echo "Pronto. Se usa kiwix-serve, reinicie para ele carregar os ZIMs novos."
