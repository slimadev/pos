package config

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// AppConfig é o espelho em disco da configuração gravada em `configuracoes`
// (SDD secção 9.3) — non-autoritativo em runtime (o SQLite é, porque
// qualquer service já tem a ligação aberta), mas é o que outra ferramenta
// (ex: um instalador, ou suporte técnico) consegue ler sem tocar na base.
type AppConfig struct {
	NegocioNome   string    `json:"negocio_nome"`
	TipoNegocio   string    `json:"tipo_negocio"`
	ModulosAtivos []string  `json:"modulos_ativos"`
	Tema          string    `json:"tema"`
	InstaladoEm   time.Time `json:"instalado_em"`
}

// LoadAppConfig lê config.json. ok=false (sem erro) significa "primeira
// execução, ficheiro ainda não existe" — não confundir com falha de leitura.
func LoadAppConfig() (cfg AppConfig, ok bool, err error) {
	path, err := ResolveConfigPath()
	if err != nil {
		return AppConfig{}, false, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AppConfig{}, false, nil
	}
	if err != nil {
		return AppConfig{}, false, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, false, err
	}
	return cfg, true, nil
}

func SaveAppConfig(cfg AppConfig) error {
	path, err := ResolveConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
