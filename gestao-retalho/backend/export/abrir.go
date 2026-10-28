package export

// abridor substitui a abertura pelo sistema operativo quando definido — no
// Android não há "programa predefinido": a app nativa abre o ficheiro com o
// visualizador/partilha do Android (DefinirAbridor).
var abridor func(caminho string) error

// DefinirAbridor troca a forma de abrir ficheiros (nil volta ao sistema).
func DefinirAbridor(f func(caminho string) error) {
	abridor = f
}

// AbrirFicheiro abre o ficheiro (recibo PDF, exportação Excel/PDF) no
// programa predefinido do sistema — no Windows, o leitor de PDF, de onde
// se imprime na impressora de recibos. Não passa pelo BrowserOpenURL do
// Wails, que recusa endereços file:// (só aceita http/https), por isso o
// ficheiro era gerado mas nunca chegava a abrir.
func AbrirFicheiro(caminho string) error {
	if abridor != nil {
		return abridor(caminho)
	}
	return abrirFicheiro(caminho)
}
