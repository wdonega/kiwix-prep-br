package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Leitura mínima do formato ZIM (https://wiki.openzim.org/wiki/ZIM_file_format): só o cabeçalho.
// Ele basta para saber se um arquivo é mesmo um ZIM, que tamanho ele deve ter e onde fica o
// checksum interno, o que permite validar um link antes de baixar e conferir o arquivo depois.

const (
	tamanhoCabecalhoZim = 80
	magicZim            = 0x044D495A // "ZIM\x04"
)

type cabecalhoZim struct {
	Entradas    uint32 // tudo o que o arquivo guarda (páginas, imagens, metadados)
	PosChecksum uint64 // posição do MD5 de tudo o que vem antes dele; são os últimos 16 bytes
}

func lerCabecalhoZim(b []byte) (cabecalhoZim, error) {
	if len(b) < tamanhoCabecalhoZim || binary.LittleEndian.Uint32(b[0:4]) != magicZim {
		return cabecalhoZim{}, fmt.Errorf("não é um arquivo ZIM")
	}
	return cabecalhoZim{
		Entradas:    binary.LittleEndian.Uint32(b[24:28]),
		PosChecksum: binary.LittleEndian.Uint64(b[72:80]),
	}, nil
}

// tamanhoDoArquivo é o tamanho que o ZIM completo tem que ter.
func (c cabecalhoZim) tamanhoDoArquivo() int64 { return int64(c.PosChecksum) + md5.Size }

// conferirZim confirma que o arquivo é um ZIM inteiro e sem corrupção, pelo checksum que ele mesmo carrega.
func conferirZim(ctx context.Context, caminho string) error {
	f, err := os.Open(caminho)
	if err != nil {
		return err
	}
	defer f.Close()
	cab := make([]byte, tamanhoCabecalhoZim)
	if _, err := io.ReadFull(f, cab); err != nil {
		return fmt.Errorf("não é um arquivo ZIM")
	}
	c, err := lerCabecalhoZim(cab)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() != c.tamanhoDoArquivo() {
		return fmt.Errorf("arquivo ZIM incompleto: tem %s e deveria ter %s", humano(info.Size()), humano(c.tamanhoDoArquivo()))
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	h := md5.New()
	buf := make([]byte, 1<<20)
	falta := int64(c.PosChecksum)
	for falta > 0 {
		if ctx.Err() != nil {
			return errCancelado
		}
		n, err := f.Read(buf[:min(int64(len(buf)), falta)])
		h.Write(buf[:n])
		falta -= int64(n)
		if err != nil && falta > 0 {
			return err
		}
	}
	guardado := make([]byte, md5.Size)
	if _, err := io.ReadFull(f, guardado); err != nil {
		return err
	}
	if !bytes.Equal(h.Sum(nil), guardado) {
		return fmt.Errorf("arquivo ZIM corrompido (o checksum interno não confere)")
	}
	return nil
}
