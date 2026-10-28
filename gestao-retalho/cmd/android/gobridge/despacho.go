package gobridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"reflect"
)

var tipoErro = reflect.TypeOf((*error)(nil)).Elem()

// despachar chama o método exportado `metodo` de `alvo` por reflexão, com a
// mesma semântica dos bindings do Wails: argumentos JSON posicionais (os
// que faltarem ficam com o valor zero), resultado = 1.º valor de retorno em
// JSON, erro = último valor de retorno quando é `error`.
func despachar(alvo any, metodo, argsJSON string) (resultado string, err error) {
	if !token.IsExported(metodo) || metodo == "OnStartup" {
		return "", fmt.Errorf("método não disponível: %q", metodo)
	}
	m := reflect.ValueOf(alvo).MethodByName(metodo)
	if !m.IsValid() {
		return "", fmt.Errorf("método desconhecido: %q", metodo)
	}
	t := m.Type()
	if t.IsVariadic() {
		return "", fmt.Errorf("método variádico não suportado: %q", metodo)
	}

	var brutos []json.RawMessage
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &brutos); err != nil {
			return "", fmt.Errorf("argumentos de %s: %w", metodo, err)
		}
	}
	if len(brutos) > t.NumIn() {
		return "", fmt.Errorf("%s recebe %d argumento(s), vieram %d", metodo, t.NumIn(), len(brutos))
	}
	args := make([]reflect.Value, t.NumIn())
	for i := range args {
		v := reflect.New(t.In(i))
		if i < len(brutos) && string(brutos[i]) != "null" {
			if err := json.Unmarshal(brutos[i], v.Interface()); err != nil {
				return "", fmt.Errorf("argumento %d de %s: %w", i+1, metodo, err)
			}
		}
		args[i] = v.Elem()
	}

	// Um panic num serviço não pode deitar abaixo a app inteira.
	defer func() {
		if r := recover(); r != nil {
			resultado, err = "", fmt.Errorf("erro interno em %s: %v", metodo, r)
		}
	}()
	saidas := m.Call(args)

	if n := len(saidas); n > 0 && t.Out(n-1) == tipoErro {
		if e := saidas[n-1]; !e.IsNil() {
			return "", e.Interface().(error)
		}
		saidas = saidas[:n-1]
	}
	if len(saidas) == 0 {
		return "null", nil
	}
	j, err := json.Marshal(saidas[0].Interface())
	if err != nil {
		return "", errors.Join(fmt.Errorf("resultado de %s", metodo), err)
	}
	return string(j), nil
}
