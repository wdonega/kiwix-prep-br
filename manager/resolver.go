package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const kiwixDownload = "https://download.kiwix.org/zim/"

// buscar faz um GET simples e devolve o corpo (com limite de tamanho).
func (a *App) buscar(ctx context.Context, url string, limite int64, cab map[string]string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limite))
}

// ---------------------------------------------------------------- Kiwix

type metalink struct {
	Arquivos []struct {
		Nome    string `xml:"name,attr"`
		Tamanho int64  `xml:"size"`
		Hashes  []struct {
			Tipo  string `xml:"type,attr"`
			Valor string `xml:",chardata"`
		} `xml:"hash"`
		URLs []struct {
			Prioridade int    `xml:"priority,attr"`
			Valor      string `xml:",chardata"`
		} `xml:"url"`
	} `xml:"file"`
}

// listagens guarda a página de índice de cada pasta do Kiwix durante uma verificação: cada pasta é
// baixada uma vez só, mesmo com vários itens dela sendo resolvidos ao mesmo tempo.
type listagens struct {
	mu     sync.Mutex
	pastas map[string]*listagem
}

type listagem struct {
	uma  sync.Once
	html string
	err  error
}

func novasListagens() *listagens { return &listagens{pastas: map[string]*listagem{}} }

func (ls *listagens) da(ctx context.Context, a *App, pasta string) (string, error) {
	ls.mu.Lock()
	l := ls.pastas[pasta]
	if l == nil {
		l = &listagem{}
		ls.pastas[pasta] = l
	}
	ls.mu.Unlock()
	l.uma.Do(func() {
		b, err := a.buscar(ctx, kiwixDownload+pasta+"/", 32<<20, nil)
		l.html, l.err = string(b), err
	})
	return l.html, l.err
}

// resolverKiwix acha a edição mais nova na listagem da pasta e lê o .meta4 dela
// (tamanho, sha-256 e lista de mirrors com prioridade).
func (a *App) resolverKiwix(ctx context.Context, it *ItemCatalogo, ls *listagens) (*Disponivel, error) {
	html, err := ls.da(ctx, a, it.Pasta)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`href="` + regexp.QuoteMeta(it.ID) + `_(\d{4}-\d{2}[a-z]?)\.zim"`)
	var edicao string
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		if m[1] > edicao {
			edicao = m[1]
		}
	}
	if edicao == "" {
		return nil, fmt.Errorf("nenhuma edição de %s em %s", it.ID, kiwixDownload+it.Pasta+"/")
	}
	nome := it.ID + "_" + edicao + ".zim"
	direto := kiwixDownload + it.Pasta + "/" + nome

	b, err := a.buscar(ctx, direto+".meta4", 8<<20, nil)
	if err != nil {
		return nil, fmt.Errorf("metalink: %w", err)
	}
	var ml metalink
	if err := xml.Unmarshal(b, &ml); err != nil || len(ml.Arquivos) == 0 {
		return nil, fmt.Errorf("metalink inválido para %s", nome)
	}
	f := ml.Arquivos[0]
	d := &Disponivel{Arquivo: nome, Edicao: edicao, Tamanho: f.Tamanho, VerificadoEm: time.Now().UTC()}
	for _, h := range f.Hashes {
		if h.Tipo == "sha-256" {
			d.SHA256 = strings.ToLower(strings.TrimSpace(h.Valor))
		}
	}
	sort.SliceStable(f.URLs, func(i, j int) bool { return f.URLs[i].Prioridade < f.URLs[j].Prioridade })
	for _, u := range f.URLs {
		if s := strings.TrimSpace(u.Valor); strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
			d.URLs = append(d.URLs, s)
		}
	}
	d.URLs = append(d.URLs, direto) // por último, o redirecionador oficial
	if d.SHA256 == "" || d.Tamanho <= 0 {
		return nil, fmt.Errorf("metalink de %s sem tamanho ou sha-256", nome)
	}
	return d, nil
}

// ---------------------------------------------------------------- GitHub

type releaseGitHub struct {
	Assets []struct {
		Nome       string    `json:"name"`
		Tamanho    int64     `json:"size"`
		Atualizado time.Time `json:"updated_at"`
		URL        string    `json:"browser_download_url"`
	} `json:"assets"`
}

func (a *App) resolverGitHub(ctx context.Context, it *ItemCatalogo) (*Disponivel, error) {
	api := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", it.Repo, it.Tag)
	b, err := a.buscar(ctx, api, 4<<20, map[string]string{"Accept": "application/vnd.github+json"})
	if err != nil {
		return nil, err
	}
	var rel releaseGitHub
	if err := json.Unmarshal(b, &rel); err != nil {
		return nil, err
	}
	var d *Disponivel
	var urlSHA string
	for _, as := range rel.Assets {
		switch as.Nome {
		case it.Arquivo:
			d = &Disponivel{Arquivo: as.Nome, Edicao: as.Atualizado.UTC().Format("2006-01-02"),
				Tamanho: as.Tamanho, URLs: []string{as.URL}, VerificadoEm: time.Now().UTC()}
		case it.Arquivo + ".sha256":
			urlSHA = as.URL
		}
	}
	if d == nil {
		return nil, fmt.Errorf("release %s sem o arquivo %s", it.Tag, it.Arquivo)
	}
	if urlSHA != "" {
		if s, err := a.buscar(ctx, urlSHA, 4096, nil); err == nil {
			if campos := strings.Fields(string(s)); len(campos) > 0 && len(campos[0]) == 64 {
				d.SHA256 = strings.ToLower(campos[0])
			}
		}
	}
	return d, nil
}

// ---------------------------------------------------------------- verificação geral

// verificar consulta a edição mais recente de todos os itens. O que falhar mantém o último valor conhecido.
func (a *App) verificar(ctx context.Context) {
	a.mu.Lock()
	if a.verificando {
		a.mu.Unlock()
		return
	}
	a.verificando = true
	a.mu.Unlock()

	// vários itens ao mesmo tempo (o catálogo tem dezenas), sem abrir conexões demais com os servidores
	ls := novasListagens()
	novos := map[string]*Disponivel{}
	erros := map[string]string{}
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		vagas = make(chan struct{}, 8)
	)
	for _, it := range a.cat.Itens {
		wg.Add(1)
		go func(it *ItemCatalogo) {
			defer wg.Done()
			vagas <- struct{}{}
			defer func() { <-vagas }()
			var d *Disponivel
			var err error
			if it.Origem == "github" {
				d, err = a.resolverGitHub(ctx, it)
			} else {
				d, err = a.resolverKiwix(ctx, it, ls)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				erros[it.ID] = err.Error()
				logf("verificação de %s falhou: %v", it.ID, err)
				return
			}
			novos[it.ID] = d
		}(it)
	}
	wg.Wait()

	a.mu.Lock()
	defer a.mu.Unlock()
	for id, d := range novos {
		a.est.Disponiveis[id] = d
	}
	a.errosVerificacao = erros
	if len(novos) > 0 {
		a.est.VerificadoEm = time.Now().UTC()
	}
	a.verificando = false
	if err := a.est.salvar(a.caminhoEstado()); err != nil {
		logf("salvar estado: %v", err)
	}
	logf("verificação concluída: %d ok, %d com erro", len(novos), len(erros))
}
