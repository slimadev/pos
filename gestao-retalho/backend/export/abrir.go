package export

// AbrirFicheiro abre o ficheiro (recibo PDF, exportação Excel/PDF) no
// programa predefinido do sistema — no Windows, o leitor de PDF, de onde
// se imprime na impressora de recibos. Não passa pelo BrowserOpenURL do
// Wails, que recusa endereços file:// (só aceita http/https), por isso o
// ficheiro era gerado mas nunca chegava a abrir.
func AbrirFicheiro(caminho string) error {
	return abrirFicheiro(caminho)
}
