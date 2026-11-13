package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

const (
	rodadasMirrors = 3                // quantas vezes percorre a lista inteira de mirrors
	tempoOcioso    = 45 * time.Second // sem receber nada por esse tempo: troca de mirror
	folgaDisco     = 200 << 20        // espaço que sempre sobra em disco
)

// Tarefa: uma operação em andamento sobre um item (download ou remoção).
type Tarefa struct {
	ID         string
	Situacao   string // na_fila, baixando, verificando, apagando
	Total      int64
	Baixado    atomic.Int64
	Velocidade int64 // bytes/s, atualizado a cada segundo
	Mirror     string
	ctx        context.Context
	cancelar   context.CancelFunc
}

var errCancelado = errors.New("cancelado")

// enfileirar coloca o download de um item na fila.
func (a *App) enfileirar(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cat.porID[id] == nil {
		return fmt.Errorf("item desconhecido")
	}
	if a.tarefas[id] != nil {
		return fmt.Errorf("já tem uma operação em andamento")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.tarefas[id] = &Tarefa{ID: id, Situacao: "na_fila", ctx: ctx, cancelar: cancel}
	delete(a.erros, id)
	select {
	case a.fila <- id:
		return nil
	default:
		delete(a.tarefas, id)
		cancel()
		return fmt.Errorf("fila cheia")
	}
}

func (a *App) cancelar(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	t := a.tarefas[id]
	if t == nil || t.Situacao == "apagando" {
		return fmt.Errorf("nada para cancelar")
	}
	t.cancelar()
	if t.Situacao == "na_fila" {
		delete(a.tarefas, id) // o trabalhador ignora quando pegar da fila
	}
	return nil
}

// trabalhador processa a fila, um download por vez (não divide a banda entre vários).
func (a *App) trabalhador() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case id := <-a.fila:
			a.mu.Lock()
			t := a.tarefas[id]
			a.mu.Unlock()
			if t == nil || t.ctx.Err() != nil {
				continue
			}
			err := a.baixar(t)
			a.mu.Lock()
			delete(a.tarefas, id)
			switch {
			case err == nil:
				delete(a.erros, id)
			case errors.Is(err, errCancelado) || t.ctx.Err() != nil:
				if a.ctx.Err() != nil {
					logf("%s: download interrompido (o parcial fica guardado e continua no próximo pedido)", id)
				} else {
					logf("%s: download cancelado", id)
				}
			default:
				a.erros[id] = err.Error()
				logf("%s: falhou: %v", id, err)
			}
			a.mu.Unlock()
		}
	}
}

func (a *App) baixar(t *Tarefa) (err error) {
	it := a.cat.porID[t.ID]

	// edição a baixar: a da última verificação; se não houver, verifica só este item agora
	a.mu.Lock()
	disp := a.est.Disponiveis[t.ID]
	a.mu.Unlock()
	if disp == nil {
		var err error
		if it.Origem == "github" {
			disp, err = a.resolverGitHub(t.ctx, it)
		} else {
			disp, err = a.resolverKiwix(t.ctx, it, novasListagens())
		}
		if err != nil {
			return fmt.Errorf("não consegui achar a versão mais recente: %w", err)
		}
		a.mu.Lock()
		a.est.Disponiveis[t.ID] = disp
		a.mu.Unlock()
	}

	final := it.nomeArquivo(disp.Edicao)
	parcial := filepath.Join(a.cfg.Dados, final+".part")
	// cancelado pelo usuário: descarta o parcial. Se só caiu (erro de rede, container reiniciado),
	// o .part fica e o próximo download continua de onde parou.
	defer func() {
		if a.ctx.Err() == nil && (errors.Is(err, errCancelado) || t.ctx.Err() != nil) {
			os.Remove(parcial)
		}
	}()

	// espaço em disco: o que falta baixar + folga
	var jaTem int64
	if info, err := os.Stat(parcial); err == nil {
		jaTem = info.Size()
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil && disp.Tamanho > 0 {
		if precisa := disp.Tamanho - jaTem + folgaDisco; livre < precisa {
			return fmt.Errorf("espaço insuficiente: precisa de %s, tem %s livres", humano(precisa), humano(livre))
		}
	}

	a.mu.Lock()
	t.Situacao, t.Total = "baixando", disp.Tamanho
	a.mu.Unlock()
	t.Baixado.Store(jaTem)

	parar := make(chan struct{})
	go a.medirVelocidade(t, parar)
	mirror, err := a.baixarDosMirrors(t, disp, parcial)
	close(parar)
	if err != nil {
		return err
	}

	a.mu.Lock()
	t.Situacao = "verificando"
	a.mu.Unlock()
	if disp.SHA256 != "" {
		soma, err := sha256DoArquivo(t.ctx, parcial)
		if err != nil {
			return err
		}
		if soma != disp.SHA256 {
			os.Remove(parcial)
			return fmt.Errorf("arquivo corrompido (sha-256 não confere); apaguei o download, tente de novo")
		}
	}

	// pronto: renomeia, registra e atualiza a biblioteca do kiwix-serve
	if err := os.Rename(parcial, filepath.Join(a.cfg.Dados, final)); err != nil {
		return err
	}
	a.mu.Lock()
	antigo := a.est.Instalados[t.ID]
	a.est.Instalados[t.ID] = &Instalado{
		Arquivo: final, Edicao: disp.Edicao, Tamanho: disp.Tamanho, SHA256: disp.SHA256,
		BaixadoEm: time.Now().UTC(), Mirror: mirror, Origem: "gerenciador",
	}
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}
	if err := a.reconstruirBiblioteca(); err != nil {
		logf("biblioteca: %v", err)
	}
	logf("%s: instalado %s (de %s)", t.ID, final, mirror)
	if antigo != nil && antigo.Arquivo != final {
		go a.apagarArquivoDepois(antigo.Arquivo)
	}
	return nil
}

// baixarDosMirrors tenta cada mirror em ordem; se um cair ou travar, continua do mesmo byte no próximo.
func (a *App) baixarDosMirrors(t *Tarefa, d *Disponivel, parcial string) (string, error) {
	var ultimoErro error
	for rodada := 0; rodada < rodadasMirrors; rodada++ {
		for _, u := range d.URLs {
			if t.ctx.Err() != nil {
				return "", errCancelado
			}
			host := u
			if pu, err := url.Parse(u); err == nil {
				host = pu.Host
			}
			a.mu.Lock()
			t.Mirror = host
			a.mu.Unlock()
			err := a.baixarDe(t, u, parcial, d.Tamanho)
			if err == nil {
				return host, nil
			}
			if t.ctx.Err() != nil {
				return "", errCancelado
			}
			ultimoErro = fmt.Errorf("%s: %w", host, err)
			logf("%s: mirror %s falhou (%v), tentando o próximo", t.ID, host, err)
		}
		select {
		case <-t.ctx.Done():
			return "", errCancelado
		case <-time.After(5 * time.Second):
		}
	}
	return "", fmt.Errorf("nenhum mirror funcionou; último erro: %v", ultimoErro)
}

// baixarDe baixa (ou continua) o arquivo a partir de uma URL.
func (a *App) baixarDe(t *Tarefa, u, parcial string, total int64) error {
	f, err := os.OpenFile(parcial, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	inicio, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if total > 0 && inicio > total { // lixo de outra edição: recomeça
		if err := f.Truncate(0); err != nil {
			return err
		}
		inicio, _ = f.Seek(0, io.SeekStart)
	}
	if total > 0 && inicio == total {
		return nil
	}
	t.Baixado.Store(inicio)

	ctx, cancel := context.WithCancel(t.ctx)
	defer cancel()
	ocioso := time.AfterFunc(tempoOcioso, cancel)
	defer ocioso.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	if inicio > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", inicio))
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusPartialContent && inicio > 0:
		// continua de onde parou
	case resp.StatusCode == http.StatusOK:
		if inicio > 0 { // mirror não aceita continuar: recomeça do zero
			if err := f.Truncate(0); err != nil {
				return err
			}
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}
			inicio = 0
			t.Baixado.Store(0)
		}
	default:
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, 256<<10)
	for {
		n, errLer := resp.Body.Read(buf)
		if n > 0 {
			ocioso.Reset(tempoOcioso)
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			t.Baixado.Add(int64(n))
		}
		if errLer == io.EOF {
			break
		}
		if errLer != nil {
			if ctx.Err() != nil && t.ctx.Err() == nil {
				return fmt.Errorf("parado há %s sem receber dados", tempoOcioso)
			}
			return errLer
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if total > 0 && t.Baixado.Load() != total {
		return fmt.Errorf("download incompleto (%s de %s)", humano(t.Baixado.Load()), humano(total))
	}
	return nil
}

func (a *App) medirVelocidade(t *Tarefa, parar chan struct{}) {
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	antes := t.Baixado.Load()
	for {
		select {
		case <-parar:
			return
		case <-tk.C:
			agora := t.Baixado.Load()
			v := agora - antes
			if v < 0 {
				v = 0
			}
			antes = agora
			a.mu.Lock()
			// média móvel simples para o número não pular tanto
			t.Velocidade = (t.Velocidade*2 + v) / 3
			a.mu.Unlock()
		}
	}
}

func sha256DoArquivo(ctx context.Context, caminho string) (string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 1<<20)
	for {
		if ctx.Err() != nil {
			return "", errCancelado
		}
		n, err := f.Read(buf)
		h.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---------------------------------------------------------------- apagar

func (a *App) apagar(id string) error {
	a.mu.Lock()
	if a.tarefas[id] != nil {
		a.mu.Unlock()
		return fmt.Errorf("já tem uma operação em andamento")
	}
	inst := a.est.Instalados[id]
	if inst == nil {
		a.mu.Unlock()
		return fmt.Errorf("não está instalado")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.tarefas[id] = &Tarefa{ID: id, Situacao: "apagando", ctx: ctx, cancelar: cancel}
	delete(a.est.Instalados, id)
	delete(a.erros, id)
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}

	go func() {
		defer cancel()
		if err := a.reconstruirBiblioteca(); err != nil {
			logf("biblioteca: %v", err)
		}
		err := a.apagarArquivo(inst.Arquivo)
		a.mu.Lock()
		delete(a.tarefas, id)
		if err != nil {
			a.erros[id] = err.Error()
		}
		a.mu.Unlock()
		logf("%s: apagado %s", id, inst.Arquivo)
	}()
	return nil
}

// apagarArquivo espera o kiwix-serve recarregar a biblioteca (e soltar o arquivo) antes de apagar.
// No Linux daria para apagar na hora; no Windows o arquivo aberto não pode ser apagado.
func (a *App) apagarArquivo(nome string) error {
	caminho := filepath.Join(a.cfg.Dados, nome)
	time.Sleep(3 * time.Second)
	var err error
	for i := 0; i < 20; i++ {
		err = os.Remove(caminho)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("não consegui apagar %s: %v", nome, err)
}

func (a *App) apagarArquivoDepois(nome string) {
	if err := a.apagarArquivo(nome); err != nil {
		logf("%v", err)
	}
}

func humano(b int64) string {
	const k = 1000
	switch {
	case b >= k*k*k:
		return fmt.Sprintf("%.1f GB", float64(b)/(k*k*k))
	case b >= k*k:
		return fmt.Sprintf("%.0f MB", float64(b)/(k*k))
	default:
		return fmt.Sprintf("%.0f kB", float64(b)/k)
	}
}
