//go:build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

// soltarRoot cuida do caso em que o container sobe como root, que é o padrão do "docker run" sem --user.
//
// O Docker cria como root a pasta de dados que não existe (-v /pasta/nova:/data), e um processo sem
// privilégio não conseguiria escrever nela. Então, só nesse instante inicial, o gerenciador usa o root
// para criar a pasta e acertar o dono, e em seguida passa a ser um usuário comum. O gerenciador e o
// kiwix-serve nunca atendem ninguém como root.
//
// Quem vira o dono:
//   - PREP_UID / PREP_GID, se informados;
//   - senão, o dono atual da pasta, quando ela já existe e não é do root (a pasta é sua: nada muda nela);
//   - senão, 1000:1000, o primeiro usuário comum da maioria das máquinas.
//
// Quando o container já sobe sem ser root (compose.yaml, --user), não faz nada.
func soltarRoot(dados string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	if err := os.MkdirAll(dados, 0o755); err != nil {
		return err
	}
	info, err := os.Stat(dados)
	if err != nil {
		return err
	}
	dono := info.Sys().(*syscall.Stat_t)
	uid, gid := 1000, 1000
	if dono.Uid != 0 {
		uid, gid = int(dono.Uid), int(dono.Gid)
	}
	if v, err := strconv.Atoi(os.Getenv("PREP_UID")); err == nil && v > 0 {
		uid = v
	}
	if v, err := strconv.Atoi(os.Getenv("PREP_GID")); err == nil && v > 0 {
		gid = v
	}
	// só a pasta em si: é o que basta para criar, trocar e apagar arquivos dentro dela
	if int(dono.Uid) != uid || int(dono.Gid) != gid {
		if err := os.Chown(dados, uid, gid); err != nil {
			return fmt.Errorf("não consegui passar a pasta %s para o usuário %d:%d: %w", dados, uid, gid, err)
		}
	}
	if err := syscall.Setgroups([]int{gid}); err != nil {
		return fmt.Errorf("largando o root (grupos): %w", err)
	}
	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("largando o root (grupo %d): %w", gid, err)
	}
	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("largando o root (usuário %d): %w", uid, err)
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("continuo como root depois de trocar de usuário")
	}
	logf("iniciado como root: pasta %s pronta, rodando agora como usuário %d:%d", dados, uid, gid)
	return nil
}
