---
titulo: Calculadora de ração para tilápias
secao: Ferramentas
---
# Calculadora de ração para tilápias

Estimativa simples baseada nas recomendações do [Documentos 259](/tanque/alimentacao): **3% do peso vivo por dia**, dividido em **pelo menos 2 tratos**. Confira sempre se sobra ração **10 minutos** depois de alimentar – se sobrar, diminua.

<div class="calc" markdown="0">
  <label>Número de peixes <input type="number" id="c-n" value="150" min="1"></label>
  <label>Peso médio de cada peixe (g) <input type="number" id="c-p" value="50" min="0.1" step="0.1"></label>
  <label>Percentual do peso vivo (%) <input type="number" id="c-pct" value="3" min="0.5" max="10" step="0.5"></label>
  <label>Tratos por dia <input type="number" id="c-t" value="2" min="1" max="8"></label>
  <div id="c-out" class="calc-out"></div>
</div>

## Referência rápida pela semana de criação (150 tilápias)

<div class="calc" markdown="0">
  <label>Semana de criação (1–14) <input type="number" id="s-w" value="1" min="1" max="14"></label>
  <div id="s-out" class="calc-out"></div>
</div>

<script src="/assets/calculadora.js"></script>

<noscript><p>Esta calculadora precisa de JavaScript. Use as <a href="/tanque/alimentacao">tabelas de alimentação</a>.</p></noscript>

!!! note "Lembretes"
    - Tanque padrão: **150 tilápias**, capacidade de suporte de **~30 kg**.
    - Consumo total de um ciclo: **~36 a 40 kg** de ração (até 50 kg / 2 sacos na prática).
    - Use ração comercial para tilápias ou peixes onívoros.
