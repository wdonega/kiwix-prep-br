package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// arquivosWeb devolve as páginas e os arquivos estáticos: embutidos no binário ou,
// em desenvolvimento, lidos de uma pasta (PREP_WEB) para não precisar recompilar a cada ajuste.
func arquivosWeb() fs.FS {
	if dir := os.Getenv("PREP_WEB"); dir != "" {
		logf("páginas servidas de %s (modo desenvolvimento)", dir)
		return os.DirFS(dir)
	}
	sub, _ := fs.Sub(webFS, "web")
	return sub
}

// pagina serve um HTML da raiz de web/ (home.html, admin.html).
func pagina(web fs.FS, nome string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(web, nome)
		if err != nil {
			http.Error(w, "página não encontrada", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(b)
	}
}

// estaticos serve /prep/ (CSS, JS, ícones). Sem cache longo: os arquivos mudam a cada versão do gerenciador.
func estaticos(web fs.FS) http.Handler {
	sub, _ := fs.Sub(web, "prep")
	servidor := http.StripPrefix("/prep/", http.FileServer(http.FS(sub)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		servidor.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------- proxy para o kiwix-serve

// O gerenciador fica na frente do kiwix-serve, então pode ajustar as páginas dele no caminho.
// Nas duas páginas com interface própria do Kiwix (barra do leitor e resultados de busca)
// entra uma folha de estilo nossa, para acompanharem o tema claro/escuro do resto.
var paginasDoKiwix = map[string]bool{"/viewer": true, "/search": true}

const folhaKiwix = `<link rel="stylesheet" href="/prep/kiwix.css">`

// O Kiwix escolhe o idioma da própria interface pelo localStorage ("userlang") ou, na falta, pelo navegador.
// Como o seletor de idioma dele fica escondido, o leitor fixa português antes de os scripts do Kiwix rodarem
// (textos da busca, dicas dos botões e a página de resultados, que o servidor só traduz para "pt").
const idiomaKiwix = `<script>try{localStorage.setItem("userlang","pt")}catch(e){}</script>`

func (a *App) proxyKiwix() *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(a.urlKiwix)
	diretor := proxy.Director
	proxy.Director = func(r *http.Request) {
		diretor(r)
		if paginasDoKiwix[r.URL.Path] {
			r.Header.Del("Accept-Encoding") // corpo sem compressão, para inserir a folha de estilo
			r.Header.Del("If-None-Match")   // senão viria um 304 da versão sem a folha de estilo
			r.Header.Del("If-Modified-Since")
		}
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		// além das duas páginas, os erros (400, 404…) também são páginas geradas pelo próprio Kiwix
		daPagina := paginasDoKiwix[resp.Request.URL.Path] && resp.StatusCode == http.StatusOK
		if !(daPagina || resp.StatusCode >= 400) || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			return nil
		}
		// o conteúdo dos acervos passa comprimido direto para o navegador; uma página de erro pode
		// vir comprimida também, e aí é aberta aqui (e segue sem compressão, são poucos bytes)
		bruto, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		comoVeio := func() error { // não deu para abrir: a página segue intacta
			resp.Body = io.NopCloser(bytes.NewReader(bruto))
			return nil
		}
		corpo := bruto
		switch resp.Header.Get("Content-Encoding") {
		case "":
		case "gzip":
			gz, err := gzip.NewReader(bytes.NewReader(bruto))
			if err != nil {
				return comoVeio()
			}
			if corpo, err = io.ReadAll(io.LimitReader(gz, 16<<20)); err != nil {
				return comoVeio()
			}
			resp.Header.Del("Content-Encoding")
		default:
			return comoVeio()
		}
		// a página 404 do Kiwix tem desenho próprio (error.css); só as de erro "cruas" recebem a nossa folha
		comDesenho := !daPagina && bytes.Contains(corpo, []byte(`rel="Stylesheet"`))
		if i := bytes.Index(corpo, []byte("</head>")); i >= 0 && !comDesenho {
			corpo = append(corpo[:i:i], append([]byte(folhaKiwix), corpo[i:]...)...)
		}
		if resp.Request.URL.Path == "/viewer" {
			if i := bytes.Index(corpo, []byte("<head>")); i >= 0 {
				i += len("<head>")
				corpo = append(corpo[:i:i], append([]byte(idiomaKiwix), corpo[i:]...)...)
			}
		}
		resp.Body = io.NopCloser(bytes.NewReader(corpo))
		resp.ContentLength = int64(len(corpo))
		resp.Header.Set("Content-Length", strconv.Itoa(len(corpo)))
		resp.Header.Del("ETag") // o ETag é do corpo original; sem ele o navegador não recebe 304 da versão sem estilo
		resp.Header.Set("Cache-Control", "no-cache")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "O Kiwix ainda está iniciando. Tente de novo em alguns segundos.", http.StatusBadGateway)
	}
	return proxy
}

// ---------------------------------------------------------------- biblioteca (dados da página inicial)

type livroXML struct {
	ID      string `xml:"id,attr"`
	Caminho string `xml:"path,attr"`
	Idioma  string `xml:"language,attr"`
	Artigos int64  `xml:"articleCount,attr"`
	Midias  int64  `xml:"mediaCount,attr"`
}

type livro struct {
	ID        string   `json:"id"`
	Titulo    string   `json:"titulo"`
	Descricao string   `json:"descricao"`
	Grupo     string   `json:"grupo"`
	Tema      string   `json:"tema,omitempty"`
	Nome      string   `json:"nome"`  // nome do acervo nas URLs do Kiwix (/content/<nome>/, /viewer#<nome>)
	Icone     string   `json:"icone"` // ilustração de 48 px servida pelo Kiwix
	Idiomas   []string `json:"idiomas"`
	Artigos   int64    `json:"artigos"`
	Midias    int64    `json:"midias"`
	Tamanho   int64    `json:"tamanho"`
	Edicao    string   `json:"edicao"`
}

// apiBiblioteca lista o que o kiwix-serve está servindo agora: os itens instalados que entraram
// no library.xml, com o título e a descrição do catálogo e os dados que o Kiwix leu de cada ZIM.
func (a *App) apiBiblioteca(w http.ResponseWriter, r *http.Request) {
	var bib struct {
		Livros []livroXML `xml:"book"`
	}
	a.muBiblioteca.Lock()
	b, err := os.ReadFile(a.caminhoBiblioteca())
	a.muBiblioteca.Unlock()
	if err == nil {
		err = xml.Unmarshal(b, &bib)
	}
	if err != nil {
		logf("lendo library.xml: %v", err)
	}
	porArquivo := map[string]livroXML{}
	for _, l := range bib.Livros {
		porArquivo[filepath.Base(l.Caminho)] = l
	}

	livros := []livro{}
	a.mu.Lock()
	for _, it := range a.cat.Itens {
		inst := a.est.Instalados[it.ID]
		if inst == nil {
			continue
		}
		x, ok := porArquivo[inst.Arquivo]
		if !ok { // instalado mas fora da biblioteca (arquivo inválido): não aparece
			continue
		}
		l := livro{
			ID: it.ID, Titulo: it.Titulo, Descricao: it.Descricao, Grupo: it.Grupo, Tema: it.Tema,
			Nome:    strings.TrimSuffix(inst.Arquivo, ".zim"),
			Icone:   "/catalog/v2/illustration/" + x.ID + "/?size=48",
			Artigos: x.Artigos, Midias: x.Midias, Tamanho: inst.Tamanho, Edicao: inst.Edicao,
			Idiomas: []string{},
		}
		for _, i := range strings.Split(x.Idioma, ",") {
			if i = strings.TrimSpace(i); i != "" {
				l.Idiomas = append(l.Idiomas, i)
			}
		}
		livros = append(livros, l)
	}
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"grupos": a.cat.Grupos, "livros": livros})
}
