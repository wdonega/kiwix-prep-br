//go:build !linux

package main

// Fora do Linux (app de desktop) o gerenciador roda como o usuário que o abriu: não há root para largar.
func soltarRoot(dados string) error { return nil }
