// prep-manager: gerencia os ZIMs do prep-wiki-br e serve tudo por uma porta só.
//
//	/        → página inicial: os acervos instalados
//	/admin/  → gerenciador (lista curada, baixar, apagar)
//	/prep/   → CSS, JS e ícones das duas páginas
//	/api/    → API JSON usada por elas
//	resto    → kiwix-serve, que roda como processo filho
//
// Pensado para rodar igual dentro do Docker e, depois, empacotado como app de desktop:
// nada de caminho fixo, binários do Kiwix achados ao lado do executável ou no PATH,
// e o kiwix-serve é iniciado e vigiado por aqui (não por script de shell).
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// versao é preenchida no build da imagem (-ldflags "-X main.versao=AAAA-MM-DD-rN"), no mesmo formato
// das releases das wikis. Num build local fica "dev".
var versao = "dev"

const userAgent = "prep-wiki-br-manager (+https://github.com/wdonega/prep-wiki-br)"

//go:embed web
var webFS embed.FS

type Config struct {
	Dados       string   // pasta dos ZIMs, do library.xml e do estado.json
	Endereco    string   // onde o gerenciador escuta
	KiwixServe  string   // caminho do kiwix-serve
	KiwixManage string   // caminho do kiwix-manage
	KiwixPorta  int      // porta interna do kiwix-serve (só 127.0.0.1)
	KiwixArgs   []string // opções extras para o kiwix-serve
}

type App struct {
	cfg      Config
	base     *Catalogo                // o catálogo embutido, fixo
	catAtual atomic.Pointer[Catalogo] // base + os acervos da pessoa; trocado inteiro quando eles mudam
	http     *http.Client
	httpFora *http.Client // para endereços informados pela pessoa: não alcança a própria máquina
	urlKiwix *url.URL
	ctx      context.Context

	mu               sync.Mutex
	est              *Estado
	tarefas          map[string]*Tarefa
	erros            map[string]string // último erro de download/remoção por item
	errosVerificacao map[string]string
	verificando      bool

	muBiblioteca sync.Mutex
	fila         chan string
	kiwixParou   chan struct{}
}

// cat devolve o catálogo em uso agora.
func (a *App) cat() *Catalogo { return a.catAtual.Load() }

// remontarCatalogo refaz o catálogo depois de uma mudança nos acervos da pessoa. Chame com a.mu travado.
func (a *App) remontarCatalogo() { a.catAtual.Store(comPersonalizados(a.base, a.est.Personalizados)) }

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

// acharBinario: valor explícito > ao lado do executável (app de desktop) > PATH.
func acharBinario(nome, explicito string) string {
	if explicito != "" {
		return explicito
	}
	if runtime.GOOS == "windows" {
		nome += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		c := filepath.Join(filepath.Dir(exe), nome)
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath(nome); err == nil {
		return p
	}
	return nome
}

func portaLivre() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 8081
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func lerConfig() Config {
	c := Config{
		Dados:       env("PREP_DADOS", "dados"),
		Endereco:    env("PREP_ENDERECO", "127.0.0.1:8080"), // padrão seguro para desktop; no Docker é 0.0.0.0
		KiwixServe:  acharBinario("kiwix-serve", os.Getenv("PREP_KIWIX_SERVE")),
		KiwixManage: acharBinario("kiwix-manage", os.Getenv("PREP_KIWIX_MANAGE")),
	}
	if p, err := strconv.Atoi(os.Getenv("PREP_KIWIX_PORTA")); err == nil && p > 0 {
		c.KiwixPorta = p
	} else {
		c.KiwixPorta = portaLivre()
	}
	if extra := strings.Fields(os.Getenv("PREP_KIWIX_ARGS")); len(extra) > 0 {
		c.KiwixArgs = extra
	}
	if abs, err := filepath.Abs(c.Dados); err == nil {
		c.Dados = abs
	}
	return c
}

func logf(formato string, args ...any) { log.Printf(formato, args...) }

func main() {
	cfg := lerConfig()
	if err := soltarRoot(cfg.Dados); err != nil { // antes de qualquer outra coisa
		log.Fatal(err)
	}
	cat, err := carregarCatalogo()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.Dados, 0o755); err != nil {
		log.Fatal(err)
	}
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	a := &App{
		cfg: cfg, base: cat, ctx: ctx,
		httpFora: clienteDeFora(),
		http:     &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: 60 * time.Second, IdleConnTimeout: 90 * time.Second}},
		urlKiwix: &url.URL{Scheme: "http", Host: "127.0.0.1:" + strconv.Itoa(cfg.KiwixPorta)},
		tarefas:  map[string]*Tarefa{}, erros: map[string]string{}, errosVerificacao: map[string]string{},
		fila: make(chan string, 100), kiwixParou: make(chan struct{}),
	}
	if a.est, err = carregarEstado(a.caminhoEstado()); err != nil {
		log.Fatalf("estado.json: %v", err)
	}
	a.remontarCatalogo()
	a.adotarArquivosSoltos()
	if err := a.est.sincronizarComDisco(cfg.Dados, a.cat()); err != nil {
		log.Fatal(err)
	}
	if err := a.est.salvar(a.caminhoEstado()); err != nil {
		log.Fatal(err)
	}
	a.limparParciaisOrfaos()
	if err := a.reconstruirBiblioteca(); err != nil {
		logf("biblioteca: %v (o kiwix-serve vai subir sem acervos)", err)
		os.WriteFile(a.caminhoBiblioteca(), []byte(bibliotecaVazia), 0o644)
	}

	go a.supervisionarKiwix()
	go a.trabalhador()
	go a.verificar(ctx)
	a.esperarKiwix(ctx)

	srv := &http.Server{Addr: cfg.Endereco, Handler: a.rotas(), ReadHeaderTimeout: 15 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(c)
	}()
	logf("dados em %s · catálogo v%d com %d ZIMs · %d da pessoa · %d instalados", cfg.Dados, cat.Versao, len(cat.Itens),
		len(a.est.Personalizados), len(a.est.Instalados))
	logf("Prep Wiki BR %s · gerenciador em http://%s/admin/", versao, cfg.Endereco)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-a.kiwixParou
}

// limparParciaisOrfaos apaga .part de edições que não são mais a mais recente conhecida.
func (a *App) limparParciaisOrfaos() {
	validos := map[string]bool{}
	for id, d := range a.est.Disponiveis {
		if it := a.cat().porID[id]; it != nil {
			validos[it.nomeArquivo(d.Edicao)+".part"] = true
		}
	}
	entradas, _ := os.ReadDir(a.cfg.Dados)
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".zim.part") && !validos[e.Name()] {
			os.Remove(filepath.Join(a.cfg.Dados, e.Name()))
			logf("apagado download incompleto antigo: %s", e.Name())
		}
	}
}

// ---------------------------------------------------------------- HTTP

func (a *App) rotas() http.Handler {
	web := arquivosWeb()
	mux := http.NewServeMux()

	// páginas próprias
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		// primeira vez, sem nada instalado: vai direto para o gerenciador
		a.mu.Lock()
		vazio := len(a.est.Instalados) == 0
		a.mu.Unlock()
		if vazio {
			http.Redirect(w, r, "/admin/", http.StatusFound)
			return
		}
		pagina(web, "home.html")(w, r)
	})
	mux.HandleFunc("GET /admin/{$}", pagina(web, "admin.html"))
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})
	mux.Handle("GET /prep/", estaticos(web))

	// API
	mux.HandleFunc("GET /api/biblioteca", a.apiBiblioteca)
	mux.HandleFunc("GET /api/estado", a.apiEstado)
	mux.HandleFunc("POST /api/verificar", func(w http.ResponseWriter, r *http.Request) {
		go a.verificar(a.ctx)
		responderOK(w)
	})
	mux.HandleFunc("POST /api/baixar/{id}", a.acao(a.enfileirar))
	mux.HandleFunc("POST /api/cancelar/{id}", a.acao(a.cancelar))
	mux.HandleFunc("POST /api/apagar/{id}", a.acao(a.apagar))
	mux.HandleFunc("POST /api/meus/inspecionar", a.apiInspecionarLink)
	mux.HandleFunc("POST /api/meus/link", a.apiAdicionarLink)
	mux.HandleFunc("PUT /api/meus/arquivo", a.apiEnviarArquivo)

	// todo o resto é do kiwix-serve
	mux.Handle("/", a.proxyKiwix())
	return protegerAcoes(mux)
}

// protegerAcoes: toda ação (POST) exige um cabeçalho próprio. Navegador não manda cabeçalho
// customizado de outro site sem permissão (CORS), então uma página maliciosa aberta no mesmo
// computador não consegue mandar apagar ou baixar nada. Importa no Docker e mais ainda no desktop.
func protegerAcoes(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && strings.HasPrefix(r.URL.Path, "/api/") &&
			r.Header.Get("X-Prep-Admin") != "1" {
			http.Error(w, "proibido", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (a *App) acao(f func(string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(r.PathValue("id")); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
			return
		}
		responderOK(w)
	}
}

func responderOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

type visaoItem struct {
	*ItemCatalogo
	Situacao   string      `json:"situacao"`
	Instalado  *Instalado  `json:"instalado"`
	Disponivel *Disponivel `json:"disponivel"`
	Progresso  *struct {
		Baixado    int64  `json:"baixado"`
		Total      int64  `json:"total"`
		Velocidade int64  `json:"velocidade"`
		Mirror     string `json:"mirror"`
	} `json:"progresso,omitempty"`
	Erro            string `json:"erro,omitempty"`
	ErroVerificacao string `json:"erro_verificacao,omitempty"`
}

func (a *App) apiEstado(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	itens := make([]visaoItem, 0, len(a.cat().Itens))
	for _, it := range a.cat().Itens {
		v := visaoItem{ItemCatalogo: it, Instalado: a.est.Instalados[it.ID], Disponivel: a.est.Disponiveis[it.ID],
			Erro: a.erros[it.ID], ErroVerificacao: a.errosVerificacao[it.ID]}
		if t := a.tarefas[it.ID]; t != nil {
			v.Situacao = t.Situacao
			if t.Situacao == "baixando" || t.Situacao == "verificando" {
				v.Progresso = &struct {
					Baixado    int64  `json:"baixado"`
					Total      int64  `json:"total"`
					Velocidade int64  `json:"velocidade"`
					Mirror     string `json:"mirror"`
				}{t.Baixado.Load(), t.Total, t.Velocidade, t.Mirror}
			}
		} else if v.Instalado != nil {
			v.Situacao = "instalado"
			if desatualizado(it, v.Instalado, v.Disponivel) {
				v.Situacao = "desatualizado"
			}
		} else {
			v.Situacao = "nao_instalado"
		}
		itens = append(itens, v)
	}
	resp := map[string]any{
		"versao":          versao,
		"catalogo_versao": a.cat().Versao,
		"grupos":          a.cat().Grupos,
		"verificado_em":   a.est.VerificadoEm,
		"verificando":     a.verificando,
		"itens":           itens,
	}
	a.mu.Unlock()
	if livre, total, err := espacoEmDisco(a.cfg.Dados); err == nil {
		resp["disco"] = map[string]int64{"livre": livre, "total": total}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

// desatualizado compara o instalado com a edição mais recente conhecida.
func desatualizado(it *ItemCatalogo, inst *Instalado, disp *Disponivel) bool {
	if disp == nil || it.personalizado() { // acervo da pessoa não tem edição a acompanhar
		return false
	}
	if inst.SHA256 != "" && disp.SHA256 != "" {
		return inst.SHA256 != disp.SHA256
	}
	if inst.Edicao == "" { // encontrado na pasta sem data no nome: não dá para saber
		return false
	}
	if it.Origem == "github" && len(inst.Edicao) == 7 && strings.HasPrefix(disp.Edicao, inst.Edicao) {
		return false // instalado "2026-11" (só mês) e disponível "2026-11-13": mesma edição, provavelmente
	}
	return disp.Edicao > inst.Edicao
}
