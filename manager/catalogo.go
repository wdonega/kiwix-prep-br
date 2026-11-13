package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// O catálogo base é fixo e vai embutido no binário: define QUAIS ZIMs existem.
// A edição de cada um é descoberta na hora (a mais recente) e guardada no estado local.
//
//go:embed catalogo.json
var catalogoJSON []byte

type ItemCatalogo struct {
	ID        string `json:"id"`
	Origem    string `json:"origem"` // "kiwix" ou "github"
	Pasta     string `json:"pasta,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Tag       string `json:"tag,omitempty"`
	Arquivo   string `json:"arquivo,omitempty"`
	Grupo     string `json:"grupo"`
	Tema      string `json:"tema,omitempty"` // subdivisão dentro do grupo (só para organizar a lista do gerenciador)
	Titulo    string `json:"titulo"`
	Descricao string `json:"descricao"`

	padrao *regexp.Regexp // reconhece os arquivos deste item na pasta de dados
}

// Grupo: seção em que os itens aparecem, na ordem do catálogo.
type Grupo struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	Dica   string `json:"dica,omitempty"` // ex.: "escolha uma"
}

type Catalogo struct {
	Versao int             `json:"versao"`
	Grupos []*Grupo        `json:"grupos"`
	Itens  []*ItemCatalogo `json:"zims"`
	porID  map[string]*ItemCatalogo
}

func carregarCatalogo() (*Catalogo, error) {
	var c Catalogo
	if err := json.Unmarshal(catalogoJSON, &c); err != nil {
		return nil, fmt.Errorf("catalogo.json inválido: %w", err)
	}
	c.porID = map[string]*ItemCatalogo{}
	grupos := map[string]bool{}
	for _, g := range c.Grupos {
		if g.ID == "" || g.Titulo == "" || grupos[g.ID] {
			return nil, fmt.Errorf("catálogo: grupo sem id/título ou repetido: %q", g.ID)
		}
		grupos[g.ID] = true
	}
	for _, it := range c.Itens {
		if it.ID == "" || c.porID[it.ID] != nil {
			return nil, fmt.Errorf("catálogo: id vazio ou repetido: %q", it.ID)
		}
		if !grupos[it.Grupo] {
			return nil, fmt.Errorf("catálogo: %s está num grupo que não existe: %q", it.ID, it.Grupo)
		}
		switch it.Origem {
		case "kiwix":
			if it.Pasta == "" {
				return nil, fmt.Errorf("catálogo: %s sem pasta", it.ID)
			}
			// ex.: wikipedia_pt_top_maxi_2026-10.zim, nhs.uk_en_medicines_2026-09b.zim
			it.padrao = regexp.MustCompile(`^` + regexp.QuoteMeta(it.ID) + `_(\d{4}-\d{2}[a-z]?)\.zim$`)
		case "github":
			if it.Repo == "" || it.Tag == "" || it.Arquivo == "" {
				return nil, fmt.Errorf("catálogo: %s sem repo/tag/arquivo", it.ID)
			}
			base := strings.TrimSuffix(it.Arquivo, ".zim")
			// ex.: plantas-medicinais_pt.zim, plantas-medicinais_pt_2026-11.zim, plantas-medicinais_pt_2026-11-13.zim
			it.padrao = regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `(?:_(\d{4}-\d{2}(?:-\d{2})?))?\.zim$`)
		default:
			return nil, fmt.Errorf("catálogo: %s com origem desconhecida %q", it.ID, it.Origem)
		}
		c.porID[it.ID] = it
	}
	return &c, nil
}

// edicaoDoArquivo diz se o nome do arquivo pertence ao item e qual edição ele traz ("" se não tiver no nome).
func (it *ItemCatalogo) edicaoDoArquivo(nome string) (string, bool) {
	m := it.padrao.FindStringSubmatch(nome)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// nomeArquivo é o nome com que uma edição é gravada em disco. Cada edição tem nome próprio,
// assim a nova nunca sobrescreve a antiga enquanto o kiwix-serve ainda a tem aberta
// (no Windows não dá para substituir arquivo aberto).
func (it *ItemCatalogo) nomeArquivo(edicao string) string {
	if it.Origem == "github" {
		base := strings.TrimSuffix(it.Arquivo, ".zim")
		if edicao == "" {
			return it.Arquivo
		}
		return base + "_" + edicao + ".zim"
	}
	return it.ID + "_" + edicao + ".zim"
}
