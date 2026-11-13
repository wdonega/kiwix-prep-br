package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// "Meus acervos": o que a própria pessoa acrescenta, por link para um .zim ou enviando o arquivo.
// Um arquivo .zim copiado à mão para a pasta de dados também entra aqui, na partida seguinte.

// ---------------------------------------------------------------- nomes

var (
	reNomeSeguro  = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	reEdicaoKiwix = regexp.MustCompile(`^(.+)_(\d{4}-\d{2}[a-z]?)$`) // nome_AAAA-MM, como o Kiwix publica
	reVariante    = regexp.MustCompile(`_(maxi|nopic|mini)$`)
)

// nomeSeguro devolve um nome de arquivo .zim que pode ser gravado na pasta de dados:
// sem caminho, sem caracteres estranhos.
func nomeSeguro(bruto string) (string, error) {
	nome := path.Base(strings.ReplaceAll(strings.TrimSpace(bruto), `\`, "/"))
	if !strings.HasSuffix(strings.ToLower(nome), ".zim") {
		return "", fmt.Errorf("precisa ser um arquivo .zim")
	}
	base := strings.Trim(reNomeSeguro.ReplaceAllString(nome[:len(nome)-len(".zim")], "_"), "._-")
	if base == "" {
		return "", fmt.Errorf("nome de arquivo inválido")
	}
	if len(base) > 150 {
		base = base[:150]
	}
	return base + ".zim", nil
}

func idDoArquivo(arquivo string) string { return "meu-" + strings.TrimSuffix(arquivo, ".zim") }

// tituloDoArquivo faz um título apresentável a partir do nome, até o ZIM dizer o dele.
func tituloDoArquivo(arquivo string) string {
	base := strings.TrimSuffix(arquivo, ".zim")
	if m := reEdicaoKiwix.FindStringSubmatch(base); m != nil {
		base = m[1]
	}
	return strings.ReplaceAll(base, "_", " ")
}

// ---------------------------------------------------------------- rede

// clienteDeFora é usado para endereços informados pela pessoa. Ele se recusa a conectar na própria
// máquina e em endereços de configuração automática: como o gerenciador não tem senha, sem isso
// qualquer um na rede poderia usá-lo para alcançar serviços que só escutam em localhost.
func clienteDeFora() *http.Client {
	discador := &net.Dialer{
		Timeout: 30 * time.Second,
		Control: func(_, endereco string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(endereco)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
				return fmt.Errorf("endereço não permitido: %s", host)
			}
			return nil
		},
	}
	return &http.Client{Transport: &http.Transport{
		DialContext:           discador.DialContext,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}}
}

// ---------------------------------------------------------------- inspeção de um link

// infoLink é o que dá para saber de um .zim remoto antes de baixá-lo.
type infoLink struct {
	URL         string `json:"url"`
	Arquivo     string `json:"arquivo"`
	Tamanho     int64  `json:"tamanho"`
	Titulo      string `json:"titulo"`
	Descricao   string `json:"descricao,omitempty"`
	Idioma      string `json:"idioma,omitempty"`
	Artigos     int64  `json:"artigos,omitempty"`
	Retoma      bool   `json:"retoma"`      // o servidor aceita continuar um download interrompido
	Conferencia string `json:"conferencia"` // "sha256" (publicado pelo Kiwix) ou "interna" (checksum do próprio ZIM)
	Mirrors     int    `json:"mirrors"`
	DoKiwix     bool   `json:"do_kiwix"`

	disp *Disponivel
}

// inspecionarLink valida um link para um .zim lendo só o cabeçalho do arquivo (80 bytes): confirma que é
// um ZIM, descobre o tamanho e se o servidor entrega o arquivo inteiro. Para links do download.kiwix.org
// busca ainda o que o Kiwix publica sobre o arquivo: sha-256, mirrors, título e descrição.
// Com jaNaLista, não reclama de o arquivo já estar na lista (é o caso de baixar de novo um item que já existe).
func (a *App) inspecionarLink(ctx context.Context, bruto string, jaNaLista bool) (*infoLink, error) {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("cole o endereço completo, começando com http:// ou https://")
	}
	arquivo, err := nomeSeguro(u.Path)
	if err != nil {
		return nil, fmt.Errorf("o link precisa terminar no nome de um arquivo .zim")
	}
	if dono := a.cat().donoDoArquivo(arquivo); dono != nil && !(jaNaLista && dono.personalizado()) {
		if dono.personalizado() {
			return nil, fmt.Errorf("esse arquivo já está na sua lista")
		}
		return nil, fmt.Errorf("esse acervo já está no catálogo, como “%s”", dono.Titulo)
	}

	// só os primeiros 80 bytes
	ctxCab, cancelar := context.WithTimeout(ctx, 30*time.Second)
	defer cancelar()
	req, err := http.NewRequestWithContext(ctxCab, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", tamanhoCabecalhoZim-1))
	resp, err := a.httpFora.Do(req)
	if err != nil {
		var erroURL *url.Error
		if errors.As(err, &erroURL) {
			err = erroURL.Err
		}
		if strings.Contains(err.Error(), "endereço não permitido") {
			return nil, fmt.Errorf("esse endereço não é aceito: ele aponta para a própria máquina do Prep Wiki BR")
		}
		return nil, fmt.Errorf("não consegui abrir o link: %v", err)
	}
	defer resp.Body.Close()
	info := &infoLink{URL: u.String(), Arquivo: arquivo, Titulo: tituloDoArquivo(arquivo), Conferencia: "interna", Mirrors: 1}
	var informado int64 = -1
	switch resp.StatusCode {
	case http.StatusPartialContent:
		info.Retoma = true
		if i := strings.LastIndex(resp.Header.Get("Content-Range"), "/"); i >= 0 {
			if n, err := strconv.ParseInt(resp.Header.Get("Content-Range")[i+1:], 10, 64); err == nil {
				informado = n
			}
		}
	case http.StatusOK:
		informado = resp.ContentLength
	case http.StatusNotFound:
		return nil, fmt.Errorf("não existe arquivo nesse endereço (erro 404)")
	default:
		return nil, fmt.Errorf("o servidor recusou o pedido (%s)", resp.Status)
	}
	cab := make([]byte, tamanhoCabecalhoZim)
	if _, err := io.ReadFull(resp.Body, cab); err != nil {
		return nil, fmt.Errorf("o link não entrega um arquivo ZIM")
	}
	zim, err := lerCabecalhoZim(cab)
	if err != nil {
		return nil, fmt.Errorf("o link não entrega um arquivo ZIM (pode ser uma página, e não o arquivo em si)")
	}
	info.Tamanho = zim.tamanhoDoArquivo()
	if informado > 0 && informado != info.Tamanho {
		return nil, fmt.Errorf("arquivo incompleto no servidor: ele tem %s, e o ZIM inteiro teria %s", humano(informado), humano(info.Tamanho))
	}

	edicao := ""
	base := strings.TrimSuffix(arquivo, ".zim")
	if m := reEdicaoKiwix.FindStringSubmatch(base); m != nil {
		base, edicao = m[1], m[2]
	}
	info.disp = &Disponivel{Arquivo: arquivo, Edicao: edicao, Tamanho: info.Tamanho, URLs: []string{u.String()}, VerificadoEm: time.Now().UTC()}

	// o que o Kiwix publica sobre os arquivos dele (tudo opcional: se falhar, segue só com o cabeçalho)
	if edicao != "" {
		if strings.HasSuffix(u.Hostname(), "download.kiwix.org") {
			info.DoKiwix = true
			direto := "https://download.kiwix.org" + u.EscapedPath()
			if d, err := a.lerMetalink(ctx, direto, arquivo, edicao); err == nil && d.Tamanho == info.Tamanho {
				info.disp = d
				info.Conferencia, info.Mirrors = "sha256", len(d.URLs)
			}
		}
		a.completarComCatalogoKiwix(ctx, info, base)
	}
	return info, nil
}

// completarComCatalogoKiwix busca título, descrição e idioma no catálogo público do Kiwix, pelo nome do arquivo.
func (a *App) completarComCatalogoKiwix(ctx context.Context, info *infoLink, base string) {
	ctx, cancelar := context.WithTimeout(ctx, 8*time.Second)
	defer cancelar()
	for _, nome := range []string{base, reVariante.ReplaceAllString(base, "")} {
		b, err := a.buscar(ctx, "https://library.kiwix.org/catalog/v2/entries?count=1&name="+url.QueryEscape(nome), 1<<20, nil)
		if err != nil {
			return
		}
		var feed struct {
			Entradas []struct {
				Titulo  string `xml:"title"`
				Resumo  string `xml:"summary"`
				Idioma  string `xml:"language"`
				Artigos int64  `xml:"articleCount"`
			} `xml:"entry"`
		}
		if xml.Unmarshal(b, &feed) != nil || len(feed.Entradas) == 0 {
			continue
		}
		e := feed.Entradas[0]
		if t := strings.TrimSpace(e.Titulo); t != "" {
			info.Titulo = t
		}
		info.Descricao, info.Idioma, info.Artigos, info.DoKiwix = strings.TrimSpace(e.Resumo), e.Idioma, e.Artigos, true
		return
	}
}

// ---------------------------------------------------------------- API

func responderErro(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
}

func lerURL(r *http.Request) string {
	var corpo struct {
		URL string `json:"url"`
	}
	json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&corpo)
	return corpo.URL
}

// POST /api/meus/inspecionar {url}: o que dá para saber do arquivo antes de baixar.
func (a *App) apiInspecionarLink(w http.ResponseWriter, r *http.Request) {
	info, err := a.inspecionarLink(r.Context(), lerURL(r), false)
	if err != nil {
		responderErro(w, http.StatusUnprocessableEntity, err)
		return
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil && livre < info.Tamanho+folgaDisco {
		responderErro(w, http.StatusUnprocessableEntity, fmt.Errorf("não cabe no disco: o arquivo tem %s e há %s livres", humano(info.Tamanho), humano(livre)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// POST /api/meus/link {url}: põe o acervo na lista e começa a baixar.
func (a *App) apiAdicionarLink(w http.ResponseWriter, r *http.Request) {
	info, err := a.inspecionarLink(r.Context(), lerURL(r), false)
	if err != nil {
		responderErro(w, http.StatusUnprocessableEntity, err)
		return
	}
	id := idDoArquivo(info.Arquivo)
	a.mu.Lock()
	if a.cat().porID[id] != nil {
		a.mu.Unlock()
		responderErro(w, http.StatusConflict, fmt.Errorf("esse arquivo já está na sua lista"))
		return
	}
	a.est.Personalizados[id] = &Personalizado{ID: id, Origem: "link", URL: info.URL, Arquivo: info.Arquivo,
		Titulo: info.Titulo, Descricao: info.Descricao, AdicionadoEm: time.Now().UTC()}
	a.est.Disponiveis[id] = info.disp
	a.remontarCatalogo()
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}
	if err := a.enfileirar(id); err != nil {
		responderErro(w, http.StatusConflict, err)
		return
	}
	logf("%s: acrescentado por link (%s)", id, info.URL)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id})
}

// PUT /api/meus/arquivo?nome=arquivo.zim, com o arquivo no corpo: recebe um .zim enviado pelo navegador.
func (a *App) apiEnviarArquivo(w http.ResponseWriter, r *http.Request) {
	arquivo, err := nomeSeguro(r.URL.Query().Get("nome"))
	if err != nil {
		responderErro(w, http.StatusUnprocessableEntity, err)
		return
	}
	id := idDoArquivo(arquivo)
	if dono := a.cat().donoDoArquivo(arquivo); dono != nil {
		if dono.personalizado() {
			err = fmt.Errorf("esse arquivo já está na sua lista")
		} else {
			err = fmt.Errorf("esse acervo já está no catálogo, como “%s”", dono.Titulo)
		}
		responderErro(w, http.StatusConflict, err)
		return
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil && r.ContentLength > 0 && livre < r.ContentLength+folgaDisco {
		responderErro(w, http.StatusInsufficientStorage, fmt.Errorf("não cabe no disco: o arquivo tem %s e há %s livres", humano(r.ContentLength), humano(livre)))
		return
	}
	final := filepath.Join(a.cfg.Dados, arquivo)
	parcial := final + ".part"
	f, err := os.OpenFile(parcial, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			err = fmt.Errorf("esse arquivo já está sendo enviado ou baixado")
		}
		responderErro(w, http.StatusConflict, err)
		return
	}
	n, err := io.Copy(f, r.Body)
	if errFechar := f.Close(); err == nil {
		err = errFechar
	}
	if err == nil && r.ContentLength > 0 && n != r.ContentLength {
		err = fmt.Errorf("o envio foi interrompido")
	}
	if err == nil {
		err = conferirZim(r.Context(), parcial)
	}
	if err == nil {
		err = os.Rename(parcial, final)
	}
	if err != nil {
		os.Remove(parcial)
		if r.Context().Err() != nil {
			err = fmt.Errorf("o envio foi interrompido")
		}
		responderErro(w, http.StatusUnprocessableEntity, err)
		return
	}
	agora := time.Now().UTC()
	a.mu.Lock()
	a.est.Personalizados[id] = &Personalizado{ID: id, Origem: "arquivo", Arquivo: arquivo, Titulo: tituloDoArquivo(arquivo), AdicionadoEm: agora}
	a.est.Instalados[id] = &Instalado{Arquivo: arquivo, Tamanho: n, BaixadoEm: agora, Origem: "gerenciador"}
	a.remontarCatalogo()
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}
	if err := a.reconstruirBiblioteca(); err != nil {
		logf("biblioteca: %v", err)
	}
	logf("%s: recebido por envio (%s)", id, humano(n))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "id": id})
}

// ---------------------------------------------------------------- partida e metadados

// adotarArquivosSoltos põe em "Meus acervos" os .zim que estão na pasta de dados e não são de nenhum
// item: é o que acontece quando a pessoa copia um arquivo para lá à mão. Roda na partida.
func (a *App) adotarArquivosSoltos() {
	entradas, err := os.ReadDir(a.cfg.Dados)
	if err != nil {
		return
	}
	novos := 0
	for _, e := range entradas {
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".zim" || a.cat().donoDoArquivo(e.Name()) != nil {
			continue
		}
		if seguro, err := nomeSeguro(e.Name()); err != nil || seguro != e.Name() {
			logf("%s: nome fora do padrão, não entra na lista (renomeie só com letras, números, ponto, hífen e sublinhado)", e.Name())
			continue
		}
		id := idDoArquivo(e.Name())
		a.est.Personalizados[id] = &Personalizado{ID: id, Origem: "arquivo", Arquivo: e.Name(),
			Titulo: tituloDoArquivo(e.Name()), AdicionadoEm: time.Now().UTC()}
		novos++
	}
	if novos > 0 {
		a.remontarCatalogo()
		logf("%d arquivo(s) .zim encontrados na pasta entraram em “Meus acervos”", novos)
	}
}

// atualizarDadosDosMeus troca o título provisório (tirado do nome do arquivo) pelo título e pela descrição
// que o próprio ZIM traz, depois que o Kiwix o leu. Chamado ao fim de reconstruirBiblioteca.
func (a *App) atualizarDadosDosMeus() {
	biblioteca := a.lerBiblioteca()
	a.mu.Lock()
	defer a.mu.Unlock()
	mudou := false
	for id, p := range a.est.Personalizados {
		inst := a.est.Instalados[id]
		if inst == nil {
			continue
		}
		l, ok := biblioteca[inst.Arquivo]
		if !ok {
			continue
		}
		if t := strings.TrimSpace(l.Titulo); t != "" && t != p.Titulo {
			p.Titulo, mudou = t, true
		}
		if d := strings.TrimSpace(l.Descricao); d != "" && d != p.Descricao {
			p.Descricao, mudou = d, true
		}
	}
	if mudou {
		a.remontarCatalogo()
		if err := a.est.salvar(a.caminhoEstado()); err != nil {
			logf("salvar estado: %v", err)
		}
	}
}
