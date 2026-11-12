(function(){
  function f(n,d){return n.toLocaleString('pt-BR',{maximumFractionDigits:d,minimumFractionDigits:d});}
  function v(id){return parseFloat(document.getElementById(id).value)||0;}
  function calc(){
    var n=v('c-n'),p=v('c-p'),pct=v('c-pct'),t=Math.max(1,Math.round(v('c-t')));
    var bio=n*p/1000, dia=n*p*pct/100;
    var racao = p<60 ? 'pelete de 2–3 mm com 36–40% PB' : 'pelete de 4–5 mm com 32–35% PB';
    var aviso = bio>30 ? '<p class="alerta">⚠ Biomassa acima de ~30 kg: passou da capacidade de suporte do tanque padrão. Hora de despescar (consumir) peixes.</p>' : '';
    document.getElementById('c-out').innerHTML =
      '<p>Biomassa no tanque: <b>'+f(bio,1)+' kg</b></p>'+
      '<p>Ração por dia: <b>'+f(dia,0)+' g</b> ('+t+' tratos de ~'+f(dia/t,0)+' g)</p>'+
      '<p>Ração por semana: <b>'+f(dia*7/1000,2)+' kg</b></p>'+
      '<p>Ração indicada: <b>'+racao+'</b></p>'+aviso;
  }
  var T=[[1,'Pó',2.0,33,0.23],[2,'Pó',3.8,56,0.39],[3,'Pó',6.0,90,0.63],[4,'Pó',8.5,115,0.80],
    [5,'Pelete 2–3 mm',12.5,131,0.92],[6,'Pelete 2–3 mm',20,180,1.26],[7,'Pelete 2–3 mm',30,225,1.58],
    [8,'Pelete 2–3 mm',40,300,2.10],[9,'Pelete 2–3 mm',52.5,370,2.59],[10,'Pelete 3–4 mm',70,473,3.31],
    [11,'Pelete 3–4 mm',95,570,3.99],[12,'Pelete 3–4 mm',130,741,5.19],[13,'Pelete 3–4 mm',175,919,6.43],
    [14,'Pelete 3–4 mm',225,1013,7.09]];
  function sem(){
    var w=Math.min(14,Math.max(1,Math.round(v('s-w')))), r=T[w-1], acc=0;
    for(var i=0;i<w;i++)acc+=T[i][4];
    document.getElementById('s-out').innerHTML=
      '<p>Ração: <b>'+r[1]+'</b> · peso esperado: <b>'+f(r[2],1)+' g</b></p>'+
      '<p>Por dia: <b>'+r[3]+' g</b> · nesta semana: <b>'+f(r[4],2)+' kg</b> · acumulado: <b>'+f(acc,2)+' kg</b></p>';
  }
  ['c-n','c-p','c-pct','c-t'].forEach(function(id){document.getElementById(id).addEventListener('input',calc);});
  document.getElementById('s-w').addEventListener('input',sem);
  calc();sem();
})();
