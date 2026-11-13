package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const bibliotecaVazia = `<?xml version="1.0" encoding="UTF-8"?>
<library version="20110515">
</library>
`

func (a *App) caminhoBiblioteca() string { return filepath.Join(a.cfg.Dados, "library.xml") }
func (a *App) caminhoEstado() string     { return filepath.Join(a.cfg.Dados, "estado.json") }

// reconstruirBiblioteca gera o library.xml do zero com o que está instalado e troca de forma atômica.
// O kiwix-serve roda com --monitorLibrary e recarrega sozinho em ~1-2 s, sem reiniciar.
func (a *App) reconstruirBiblioteca() error {
	a.muBiblioteca.Lock()
	defer a.muBiblioteca.Unlock()

	a.mu.Lock()
	idDoArquivo := map[string]string{}
	var arquivos []string
	for id, inst := range a.est.Instalados {
		if _, err := os.Stat(filepath.Join(a.cfg.Dados, inst.Arquivo)); err == nil {
			arquivos = append(arquivos, inst.Arquivo)
			idDoArquivo[inst.Arquivo] = id
		}
	}
	a.mu.Unlock()
	sort.Strings(arquivos)

	tmp := a.caminhoBiblioteca() + ".tmp"
	os.Remove(tmp)
	if err := os.WriteFile(tmp, []byte(bibliotecaVazia), 0o644); err != nil {
		return err
	}
	// Um arquivo por vez: se um ZIM estiver corrompido, só ele fica de fora (com aviso no admin),
	// em vez de o kiwix-manage falhar e o kiwix-serve ficar sem nenhum acervo.
	invalidos := map[string]string{}
	for _, arq := range arquivos {
		// caminho relativo: o library.xml fica na mesma pasta dos ZIMs
		cmd := exec.Command(a.cfg.KiwixManage, filepath.Base(tmp), "add", arq)
		cmd.Dir = a.cfg.Dados
		if saida, err := cmd.CombinedOutput(); err != nil {
			logf("biblioteca: %s ficou de fora: %v: %s", arq, err, strings.TrimSpace(string(saida)))
			invalidos[idDoArquivo[arq]] = "O arquivo " + arq + " está corrompido ou não é um ZIM válido e não aparece na biblioteca. Apague e baixe de novo."
		}
	}
	// O Kiwix guarda um acervo por identificador do ZIM: se dois arquivos são o mesmo ZIM com nomes
	// diferentes, só um entra, sem erro nenhum. O outro ganha um aviso, para não parecer instalado e sumido.
	naBiblioteca := lerBibliotecaDe(tmp)
	for _, arq := range arquivos {
		if _, ok := naBiblioteca[arq]; !ok && invalidos[idDoArquivo[arq]] == "" {
			invalidos[idDoArquivo[arq]] = "O arquivo " + arq + " tem o mesmo conteúdo de outro acervo já instalado, então não aparece na biblioteca. Pode apagar."
		}
	}
	a.mu.Lock()
	for id := range a.est.Instalados {
		if msg, ok := invalidos[id]; ok {
			a.erros[id] = msg
		} else if strings.HasPrefix(a.erros[id], "O arquivo ") {
			delete(a.erros, id)
		}
	}
	a.mu.Unlock()
	if err := os.Rename(tmp, a.caminhoBiblioteca()); err != nil {
		return err
	}
	a.atualizarDadosDosMeus()
	return nil
}

// supervisionarKiwix inicia o kiwix-serve como processo filho e o reinicia se ele cair.
func (a *App) supervisionarKiwix() {
	defer close(a.kiwixParou)
	espera := time.Second
	for a.ctx.Err() == nil {
		args := []string{
			"--address=127.0.0.1",
			"--port=" + strconv.Itoa(a.cfg.KiwixPorta),
			"--monitorLibrary",
		}
		args = append(args, a.cfg.KiwixArgs...)
		args = append(args, "--library", a.caminhoBiblioteca())

		cmd := exec.CommandContext(a.ctx, a.cfg.KiwixServe, args...)
		cmd.Dir = a.cfg.Dados
		cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
		cmd.WaitDelay = 5 * time.Second
		saida, _ := cmd.StdoutPipe()
		cmd.Stderr = cmd.Stdout
		inicio := time.Now()
		if err := cmd.Start(); err != nil {
			logf("kiwix-serve não iniciou: %v", err)
		} else {
			logf("kiwix-serve iniciado (pid %d, porta interna %d)", cmd.Process.Pid, a.cfg.KiwixPorta)
			go repassarLog("kiwix-serve", saida)
			err := cmd.Wait()
			if a.ctx.Err() != nil {
				return
			}
			logf("kiwix-serve parou: %v", err)
			if time.Since(inicio) > time.Minute {
				espera = time.Second
			}
		}
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(espera):
		}
		if espera < 30*time.Second {
			espera *= 2
		}
	}
}

func repassarLog(prefixo string, r io.Reader) {
	if r == nil {
		return
	}
	s := bufio.NewScanner(r)
	for s.Scan() {
		logf("[%s] %s", prefixo, s.Text())
	}
}

// esperarKiwix aguarda o kiwix-serve aceitar conexões (para o proxy não responder erro logo no início).
func (a *App) esperarKiwix(ctx context.Context) {
	for i := 0; i < 50 && ctx.Err() == nil; i++ {
		if resp, err := a.http.Get(a.urlKiwix.String() + "/"); err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
