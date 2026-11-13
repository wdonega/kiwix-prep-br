package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Instalado: o que está em disco, com os metadados salvos na hora do download.
type Instalado struct {
	Arquivo   string    `json:"arquivo"`
	Edicao    string    `json:"edicao"`
	Tamanho   int64     `json:"tamanho"`
	SHA256    string    `json:"sha256,omitempty"`
	BaixadoEm time.Time `json:"baixado_em,omitempty"`
	Mirror    string    `json:"mirror,omitempty"`
	// "gerenciador" = baixado por aqui; "encontrado" = já estava na pasta (metadados parciais)
	Origem string `json:"origem"`
}

// Disponivel: a edição mais recente encontrada na última verificação (guardada para funcionar offline).
type Disponivel struct {
	Arquivo      string    `json:"arquivo"` // nome do arquivo na origem
	Edicao       string    `json:"edicao"`
	Tamanho      int64     `json:"tamanho"`
	SHA256       string    `json:"sha256,omitempty"`
	URLs         []string  `json:"urls"` // mirrors, em ordem de preferência
	VerificadoEm time.Time `json:"verificado_em"`
}

// Personalizado: um acervo que não é do catálogo, acrescentado pela pessoa por link ou por arquivo.
type Personalizado struct {
	ID           string    `json:"id"`
	Origem       string    `json:"origem"` // "link" ou "arquivo"
	URL          string    `json:"url,omitempty"`
	Arquivo      string    `json:"arquivo"`
	Titulo       string    `json:"titulo"`
	Descricao    string    `json:"descricao,omitempty"`
	AdicionadoEm time.Time `json:"adicionado_em"`
}

type Estado struct {
	Versao         int                       `json:"versao"`
	Instalados     map[string]*Instalado     `json:"instalados"`
	Disponiveis    map[string]*Disponivel    `json:"disponiveis"`
	Personalizados map[string]*Personalizado `json:"personalizados,omitempty"`
	VerificadoEm   time.Time                 `json:"verificado_em,omitempty"`
}

func novoEstado() *Estado {
	return &Estado{Versao: 1, Instalados: map[string]*Instalado{}, Disponiveis: map[string]*Disponivel{},
		Personalizados: map[string]*Personalizado{}}
}

func carregarEstado(caminho string) (*Estado, error) {
	b, err := os.ReadFile(caminho)
	if errors.Is(err, os.ErrNotExist) {
		return novoEstado(), nil
	}
	if err != nil {
		return nil, err
	}
	e := novoEstado()
	if err := json.Unmarshal(b, e); err != nil {
		return nil, err
	}
	if e.Instalados == nil {
		e.Instalados = map[string]*Instalado{}
	}
	if e.Disponiveis == nil {
		e.Disponiveis = map[string]*Disponivel{}
	}
	if e.Personalizados == nil {
		e.Personalizados = map[string]*Personalizado{}
	}
	return e, nil
}

// salvar grava de forma atômica (arquivo temporário + rename), para não corromper se cair no meio.
func (e *Estado) salvar(caminho string) error {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	tmp := caminho + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, caminho)
}

// sincronizarComDisco ajusta o estado ao que existe de fato na pasta:
// tira o que sumiu e reconhece ZIMs do catálogo que já estavam lá (copiados à mão, por exemplo).
func (e *Estado) sincronizarComDisco(dir string, cat *Catalogo) error {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	existe := map[string]os.FileInfo{}
	for _, d := range entradas {
		if d.Type().IsRegular() && filepath.Ext(d.Name()) == ".zim" {
			if info, err := d.Info(); err == nil {
				existe[d.Name()] = info
			}
		}
	}
	for id, inst := range e.Instalados {
		if _, ok := existe[inst.Arquivo]; !ok || cat.porID[id] == nil {
			delete(e.Instalados, id)
		}
	}
	nomes := make([]string, 0, len(existe))
	for n := range existe {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	for _, it := range cat.Itens {
		if e.Instalados[it.ID] != nil {
			continue
		}
		// se houver mais de uma edição na pasta, fica a mais nova
		var melhor, edMelhor string
		for _, n := range nomes {
			if ed, ok := it.edicaoDoArquivo(n); ok && (melhor == "" || ed > edMelhor) {
				melhor, edMelhor = n, ed
			}
		}
		if melhor != "" {
			e.Instalados[it.ID] = &Instalado{
				Arquivo: melhor, Edicao: edMelhor, Tamanho: existe[melhor].Size(), Origem: "encontrado",
			}
		}
	}
	return nil
}
