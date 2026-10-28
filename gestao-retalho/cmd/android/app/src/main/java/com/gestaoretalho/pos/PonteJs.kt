package com.gestaoretalho.pos

import android.webkit.JavascriptInterface
import android.webkit.WebView
import com.gestaoretalho.gobridge.Gobridge
import org.json.JSONObject
import java.util.concurrent.Executors

/**
 * window.Ponte: recebe as chamadas da ponte JS (web/ponte.js), corre-as no
 * backend Go fora da thread da interface e devolve o resultado por
 * window.__ponte.resolver(id, ok, dados). Como no Wails, várias chamadas
 * podem correr em paralelo (o SQLite serializa as escritas).
 */
class PonteJs(private val web: WebView) {
    @JavascriptInterface
    fun chamar(id: Int, alvo: String, metodo: String, argsJson: String) {
        executor.execute {
            val (ok, dados) = try {
                val erro = RetalhoApp.erroArranque
                if (erro != null) false to "O backend não arrancou: $erro"
                else true to Gobridge.chamar(alvo, metodo, argsJson)
            } catch (e: Exception) {
                false to (e.message ?: e.toString())
            }
            // U+2028/U+2029 são válidos em JSON mas partem uma string JS no
            // Chrome 62 (antes do ES2019): escapam-se à parte.
            val literal = JSONObject.quote(dados).replace("\u2028", "\\u2028").replace("\u2029", "\\u2029")
            val js = "window.__ponte.resolver($id,$ok,$literal)"
            web.post { web.evaluateJavascript(js, null) }
        }
    }

    companion object {
        private val executor = Executors.newFixedThreadPool(4)
    }
}
