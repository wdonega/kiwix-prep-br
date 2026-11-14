package main

// Senha do gerenciador. Com PREP_SENHA definida, o admin e as ações da API pedem a senha;
// sem ela, fica tudo aberto como antes (uso em casa, desktop). A biblioteca e a leitura
// continuam sempre abertas.
//
// Depois de entrar, o navegador guarda um cookie assinado (HMAC) que vale 30 dias. A chave
// fica na pasta de dados, então reiniciar não desloga ninguém; e a assinatura inclui um
// resumo da senha, então trocar a senha desloga todo mundo.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cookieSessao  = "prep_sessao"
	duracaoSessao = 30 * 24 * time.Hour
)

type Porteiro struct {
	senha  []byte // vazio = sem senha
	chave  []byte
	muErro sync.Mutex // senha errada: uma tentativa por vez, com espera
}

func novoPorteiro(dados, senha string) (*Porteiro, error) {
	p := &Porteiro{senha: []byte(senha)}
	if senha == "" {
		return p, nil
	}
	caminho := filepath.Join(dados, ".chave-sessao")
	if b, err := os.ReadFile(caminho); err == nil && len(b) >= 32 {
		p.chave = b
		return p, nil
	}
	p.chave = make([]byte, 32)
	if _, err := rand.Read(p.chave); err != nil {
		return nil, err
	}
	if err := os.WriteFile(caminho, p.chave, 0o600); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Porteiro) ativo() bool { return len(p.senha) > 0 }

func (p *Porteiro) assinar(validade int64) string {
	resumo := sha256.Sum256(p.senha)
	m := hmac.New(sha256.New, p.chave)
	m.Write([]byte(strconv.FormatInt(validade, 10)))
	m.Write([]byte{'|'})
	m.Write(resumo[:])
	return hex.EncodeToString(m.Sum(nil))
}

// liberado: sem senha configurada, ou com um cookie válido e dentro da validade.
func (p *Porteiro) liberado(r *http.Request) bool {
	if !p.ativo() {
		return true
	}
	c, err := r.Cookie(cookieSessao)
	if err != nil {
		return false
	}
	validade, assinatura, ok := strings.Cut(c.Value, ".")
	if !ok {
		return false
	}
	v, err := strconv.ParseInt(validade, 10, 64)
	if err != nil || time.Now().Unix() > v {
		return false
	}
	return hmac.Equal([]byte(assinatura), []byte(p.assinar(v)))
}

func viaHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (p *Porteiro) gravarCookie(w http.ResponseWriter, r *http.Request, valor string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieSessao, Value: valor, Path: "/", MaxAge: maxAge,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: viaHTTPS(r),
	})
}

// POST /api/entrar {"senha": "..."}
func (p *Porteiro) apiEntrar(w http.ResponseWriter, r *http.Request) {
	if !p.ativo() {
		responderOK(w)
		return
	}
	var corpo struct {
		Senha string `json:"senha"`
	}
	json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&corpo)
	resumoDado := sha256.Sum256([]byte(corpo.Senha))
	resumoCerto := sha256.Sum256(p.senha)
	if subtle.ConstantTimeCompare(resumoDado[:], resumoCerto[:]) != 1 {
		// Uma tentativa errada por vez, cada uma esperando 1 s: chutar senha em massa fica inviável.
		p.muErro.Lock()
		time.Sleep(time.Second)
		p.muErro.Unlock()
		logf("senha errada no admin (de %s)", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"erro":"senha incorreta"}`))
		return
	}
	validade := time.Now().Add(duracaoSessao).Unix()
	p.gravarCookie(w, r, strconv.FormatInt(validade, 10)+"."+p.assinar(validade), int(duracaoSessao/time.Second))
	responderOK(w)
}

// POST /api/sair
func (p *Porteiro) apiSair(w http.ResponseWriter, r *http.Request) {
	p.gravarCookie(w, r, "", -1)
	responderOK(w)
}

// rotasAbertas: o que funciona sem senha (a biblioteca da página inicial e o próprio login).
var rotasAbertas = map[string]bool{"/api/biblioteca": true, "/api/entrar": true, "/api/sair": true}

// proteger: /admin/ sem sessão mostra a tela de senha; a API do admin responde 401.
func (p *Porteiro) proteger(h http.Handler, telaDeSenha http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.liberado(r) {
			h.ServeHTTP(w, r)
			return
		}
		switch {
		case r.URL.Path == "/admin/" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
			telaDeSenha(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/") && !rotasAbertas[r.URL.Path]:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"erro":"é preciso entrar com a senha"}`))
		default:
			h.ServeHTTP(w, r)
		}
	})
}
