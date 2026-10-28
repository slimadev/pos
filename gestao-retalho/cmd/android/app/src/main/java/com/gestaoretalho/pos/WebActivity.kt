package com.gestaoretalho.pos

import android.app.Activity
import android.app.AlarmManager
import android.app.AlertDialog
import android.webkit.JsPromptResult
import android.webkit.JsResult
import android.widget.EditText
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import org.json.JSONObject
import android.os.Bundle
import android.provider.OpenableColumns
import android.util.Log
import android.webkit.MimeTypeMap
import android.widget.Toast
import android.webkit.ConsoleMessage
import android.webkit.JavascriptInterface
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import android.graphics.Color
import android.os.Build
import android.widget.FrameLayout
import androidx.core.content.ContextCompat
import androidx.core.view.ViewCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.webkit.WebViewAssetLoader
import com.gestaoretalho.gobridge.Gobridge
import java.io.File

/**
 * Activity que hospeda um dos frontends React numa WebView. Os ficheiros vêm
 * de assets/web/ e são servidos em https://appassets.androidplatform.net/
 * para o JS correr num origin normal (não file://).
 *
 * O terminal é do PDV: com a loja configurada e operadores ativos, a app
 * abre direto no PDV (cada operador entra com o seu PIN). O admin só abre
 * pelo botão "Administração", sempre com login, e ao sair dele a sessão
 * termina e o terminal volta ao PDV. Sem loja ou sem operadores, abre no
 * admin — o PDV não vende sem operadores.
 */
abstract class WebActivity : Activity() {
    protected lateinit var web: WebView

    /** Caminho dentro de assets/web/, ex: "admin/index.html". */
    protected abstract val pagina: String

    /** true = esta Activity passou o controlo a outra e já chamou finish(). */
    protected open fun desviar(savedInstanceState: Bundle?): Boolean = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (desviar(savedInstanceState)) return
        WebView.setWebContentsDebuggingEnabled(true)

        val loader = WebViewAssetLoader.Builder()
            .addPathHandler("/", WebViewAssetLoader.AssetsPathHandler(this))
            .build()

        web = WebView(this)
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        web.webViewClient = object : WebViewClient() {
            override fun shouldInterceptRequest(view: WebView, request: WebResourceRequest): WebResourceResponse? =
                loader.shouldInterceptRequest(request.url)
        }
        // Mensagens da consola JS vão para o logcat (tag GestaoWeb).
        web.webChromeClient = object : WebChromeClient() {
            override fun onConsoleMessage(m: ConsoleMessage): Boolean {
                Log.i("GestaoWeb", "${m.messageLevel()} ${m.message()} @${m.sourceId()}:${m.lineNumber()}")
                return true
            }

            // window.alert/confirm/prompt como diálogos da app: os da WebView
            // mostram "The page at https://appassets.androidplatform.net says".
            override fun onJsAlert(v: WebView, url: String, mensagem: String, r: JsResult): Boolean {
                dialogo(mensagem).setPositiveButton("OK") { _, _ -> r.confirm() }
                    .setOnCancelListener { r.confirm() }.show()
                return true
            }

            override fun onJsConfirm(v: WebView, url: String, mensagem: String, r: JsResult): Boolean {
                dialogo(mensagem)
                    .setPositiveButton("Confirmar") { _, _ -> r.confirm() }
                    .setNegativeButton("Cancelar") { _, _ -> r.cancel() }
                    .setOnCancelListener { r.cancel() }.show()
                return true
            }

            override fun onJsPrompt(v: WebView, url: String, mensagem: String, valor: String?, r: JsPromptResult): Boolean {
                val campo = EditText(this@WebActivity).apply { setText(valor ?: ""); setSingleLine() }
                dialogo(mensagem).setView(campo)
                    .setPositiveButton("OK") { _, _ -> r.confirm(campo.text.toString()) }
                    .setNegativeButton("Cancelar") { _, _ -> r.cancel() }
                    .setOnCancelListener { r.cancel() }.show()
                return true
            }
        }
        web.addJavascriptInterface(Nativo(), "Nativo")
        web.addJavascriptInterface(PonteJs(web), "Ponte")
        setContentView(comMargensDoSistema(web))
        web.loadUrl("https://appassets.androidplatform.net/web/$pagina")
    }

    @Deprecated("Ainda é o caminho certo em API 25")
    override fun onBackPressed() {
        if (web.canGoBack()) web.goBack() else super.onBackPressed()
    }

    override fun onDestroy() {
        if (::web.isInitialized) web.destroy()
        super.onDestroy()
    }

    /**
     * Android 15+ (targetSdk 35+) desenha sempre de ponta a ponta: a página
     * ficava por baixo da barra de estado e da de navegação. Aí a WebView vai
     * dentro de um contentor com a margem dessas barras (e do teclado, que
     * deixa de redimensionar a janela sozinho). No Sunmi (Android 7) nada
     * muda: o sistema já reserva esse espaço.
     */
    private fun comMargensDoSistema(conteudo: WebView): android.view.View {
        if (Build.VERSION.SDK_INT < 35) return conteudo
        val caixa = FrameLayout(this)
        caixa.setBackgroundColor(Color.WHITE)
        caixa.addView(conteudo, FrameLayout.LayoutParams(-1, -1))
        WindowCompat.getInsetsController(window, window.decorView).apply {
            isAppearanceLightStatusBars = true
            isAppearanceLightNavigationBars = true
        }
        ViewCompat.setOnApplyWindowInsetsListener(caixa) { v, insets ->
            val barras = insets.getInsets(WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.displayCutout())
            val teclado = insets.getInsets(WindowInsetsCompat.Type.ime())
            v.setPadding(barras.left, barras.top, barras.right, maxOf(barras.bottom, teclado.bottom))
            WindowInsetsCompat.CONSUMED
        }
        return caixa
    }

    private fun dialogo(mensagem: String): AlertDialog.Builder =
        AlertDialog.Builder(this, android.R.style.Theme_DeviceDefault_Light_Dialog_Alert)
            .setMessage(mensagem)

    // --- Seletor de ficheiros (importação de produtos do Excel) -------------

    private var aoEscolher: ((String) -> Unit)? = null

    /** Abre o seletor do Android; `resposta` recebe o caminho de uma cópia
     *  local do ficheiro escolhido, ou "" se o utilizador cancelar. */
    fun escolherFicheiro(titulo: String, extensoes: String, resposta: (String) -> Unit) {
        aoEscolher?.invoke("") // um pedido anterior por responder fica cancelado
        aoEscolher = resposta
        // Pedir o tipo certo: com "*/*" o gestor de ficheiros da Sunmi abria
        // um .zip como pasta em vez de o escolher.
        // Extensões sem tipo conhecido (ex: .retalho das cópias) obrigam a
        // pedir "*/*" — senão os ficheiros nem apareciam.
        val tipos = extensoes.split(';').map { ext ->
            MimeTypeMap.getSingleton().getMimeTypeFromExtension(ext.trim().lowercase())
        }
        val pedir = Intent(Intent.ACTION_GET_CONTENT).addCategory(Intent.CATEGORY_OPENABLE)
        val conhecidos = tipos.filterNotNull().distinct()
        when {
            tipos.any { it == null } || conhecidos.isEmpty() -> pedir.setType("*/*")
            conhecidos.size == 1 -> pedir.setType(conhecidos[0])
            else -> pedir.setType("*/*").putExtra(Intent.EXTRA_MIME_TYPES, conhecidos.toTypedArray())
        }
        @Suppress("DEPRECATION")
        startActivityForResult(Intent.createChooser(pedir, titulo), PEDIDO_FICHEIRO)
    }

    private var copiaAGuardar: File? = null

    @Deprecated("Ainda é o caminho certo em API 25")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        @Suppress("DEPRECATION")
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == PEDIDO_GUARDAR) {
            val origem = copiaAGuardar
            copiaAGuardar = null
            val destino = data?.data
            if (resultCode != RESULT_OK || origem == null || destino == null) return
            val ok = try {
                contentResolver.openOutputStream(destino)!!.use { saida -> origem.inputStream().use { it.copyTo(saida) } }
                true
            } catch (e: Exception) {
                Log.e("Retalho", "guardar cópia", e)
                false
            }
            Toast.makeText(this, if (ok) "Cópia guardada." else "Não foi possível guardar a cópia.", Toast.LENGTH_LONG).show()
            return
        }
        if (requestCode != PEDIDO_FICHEIRO) return
        val resposta = aoEscolher ?: return
        aoEscolher = null
        val uri = data?.data
        if (resultCode != RESULT_OK || uri == null) {
            resposta("")
            return
        }
        // O Go lê ficheiros por caminho: copia-se o conteúdo para a cache.
        // O nome real (o URI das Transferências é só um número).
        val nome = contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { c ->
            if (c.moveToFirst()) c.getString(0) else null
        }?.substringAfterLast('/')?.takeIf { it.isNotBlank() } ?: "ficheiro-escolhido"
        val destino = File(File(cacheDir, "importar").apply { mkdirs() }, nome)
        try {
            contentResolver.openInputStream(uri)!!.use { entrada ->
                destino.outputStream().use { saida -> entrada.copyTo(saida) }
            }
            resposta(destino.absolutePath)
        } catch (e: Exception) {
            Log.e("Retalho", "copiar ficheiro escolhido", e)
            resposta("")
        }
    }

    /** Funções nativas chamadas pelo JS (window.Nativo). Correm fora da UI thread. */
    inner class Nativo {
        /** Botão "PDV" do admin: termina a sessão e volta ao PDV (que fica
         *  por baixo quando o admin foi aberto a partir dele). */
        @JavascriptInterface
        fun abrirPDV() {
            runOnUiThread {
                terminarSessaoAdmin()
                if (!intent.getBooleanExtra(EXTRA_VINDO_DO_PDV, false)) {
                    startActivity(Intent(this@WebActivity, PdvActivity::class.java))
                }
                finish()
            }
        }

        /** Botão "Administração" do PDV: abre o admin sempre no login. */
        @JavascriptInterface
        fun abrirAdmin() {
            runOnUiThread {
                terminarSessaoAdmin()
                startActivity(Intent(this@WebActivity, MainActivity::class.java).putExtra(EXTRA_VINDO_DO_PDV, true))
            }
        }

        /** "Guardar em…": o utilizador escolhe onde gravar a cópia de
         *  segurança `nome` (cartão SD, pen USB, Transferências, Drive) pelo
         *  seletor do Android — sem pedir permissões de armazenamento. */
        @JavascriptInterface
        fun guardarCopia(nome: String) {
            if (!Regex("copia-[0-9-]+\\.(retalho|zip)").matches(nome)) return
            runOnUiThread {
                copiaAGuardar = File(filesDir, "gestao-retalho-admin/copias/$nome")
                val pedir = Intent(Intent.ACTION_CREATE_DOCUMENT)
                    .addCategory(Intent.CATEGORY_OPENABLE)
                    .setType("application/octet-stream")
                    .putExtra(Intent.EXTRA_TITLE, nome)
                @Suppress("DEPRECATION")
                startActivityForResult(pedir, PEDIDO_GUARDAR)
            }
        }

        /** Reinicia a app (processo novo): o restauro de uma cópia de
         *  segurança só é aplicado no arranque, antes de abrir as bases. */
        @JavascriptInterface
        fun reiniciarApp() {
            runOnUiThread {
                val abrir = Intent(this@WebActivity, MainActivity::class.java)
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK)
                val pendente = PendingIntent.getActivity(
                    this@WebActivity, 0, abrir,
                    PendingIntent.FLAG_CANCEL_CURRENT or PendingIntent.FLAG_IMMUTABLE,
                )
                (getSystemService(Context.ALARM_SERVICE) as AlarmManager)
                    .set(AlarmManager.RTC, System.currentTimeMillis() + 400, pendente)
                finishAffinity()
                Runtime.getRuntime().exit(0)
            }
        }
    }

    // --- Leitor de códigos de barras (Sunmi) --------------------------------
    //
    // A cabeça de leitura da Sunmi entrega cada código por broadcast; aqui
    // passa para o ecrã ativo como evento `codigo-lido` (o catálogo do PDV,
    // a receção e o inventário ouvem-no). Só o ecrã visível recebe códigos.

    private val leitor = object : BroadcastReceiver() {
        override fun onReceive(c: Context, i: Intent) {
            val codigo = i.getStringExtra("data") ?: i.getStringExtra("code") ?: return
            entregarCodigo(codigo.trim())
        }
    }

    fun entregarCodigo(codigo: String) {
        if (codigo.isEmpty()) return
        val js = "window.dispatchEvent(new CustomEvent('codigo-lido',{detail:${JSONObject.quote(codigo)}}))"
        web.post { web.evaluateJavascript(js, null) }
    }

    override fun onResume() {
        super.onResume()
        // O broadcast vem de outra app (serviço de leitura da Sunmi): no
        // Android 14+ é obrigatório declarar EXPORTED, senão a app fecha.
        ContextCompat.registerReceiver(this, leitor, IntentFilter(ACAO_LEITOR_SUNMI), ContextCompat.RECEIVER_EXPORTED)
        // A WebView não recebe "focus" ao voltar de outra Activity; o PDV usa-o
        // para ver operadores criados ou desativados no admin entretanto.
        web.evaluateJavascript("window.dispatchEvent(new Event('focus'))", null)
    }

    override fun onPause() {
        unregisterReceiver(leitor)
        super.onPause()
    }
}

private const val PEDIDO_FICHEIRO = 4201
private const val PEDIDO_GUARDAR = 4202

/** Broadcast do serviço de leitura da Sunmi (com.sunmi.scanner); o código
 *  vem no extra "data". */
private const val ACAO_LEITOR_SUNMI = "com.sunmi.scanner.ACTION_DATA_CODE_RECEIVED"

/** Extra da MainActivity aberta pelo botão "Administração" do PDV. */
private const val EXTRA_VINDO_DO_PDV = "vindoDoPdv"

/** Fecha a sessão do administrador: quem pegar no terminal a seguir tem de
 *  voltar a pôr a password. */
fun terminarSessaoAdmin() {
    try {
        Gobridge.chamar("admin.AdminApp", "Logout", "[]")
    } catch (e: Exception) {
        Log.e("Retalho", "logout", e)
    }
}

class MainActivity : WebActivity() {
    override val pagina = "admin/index.html"

    /** Arranque normal (ícone): com loja e operadores vai direto ao PDV. */
    override fun desviar(savedInstanceState: Bundle?): Boolean {
        if (savedInstanceState != null || intent.getBooleanExtra(EXTRA_VINDO_DO_PDV, false)) return false
        if (!prontoParaVender()) return false
        startActivity(Intent(this, PdvActivity::class.java))
        finish()
        return true
    }

    private fun prontoParaVender(): Boolean {
        if (RetalhoApp.erroArranque != null) return false
        return try {
            val e = JSONObject(Gobridge.chamar("admin.AdminApp", "ObterEstadoArranque", "[]"))
            !e.optBoolean("PrecisaConfigurar", true) && e.optInt("OperadoresAtivos", 0) > 0
        } catch (e: Exception) {
            Log.e("Retalho", "estado de arranque", e)
            false
        }
    }

    override fun onDestroy() {
        terminarSessaoAdmin()
        super.onDestroy()
    }
}

class PdvActivity : WebActivity() {
    override val pagina = "pdv/index.html"

    /** O PDV é o ecrã principal do terminal: "voltar" põe a app em segundo
     *  plano em vez de a fechar (e nunca leva ao admin sem password). */
    @Deprecated("Ainda é o caminho certo em API 25")
    override fun onBackPressed() {
        if (web.canGoBack()) web.goBack() else moveTaskToBack(true)
    }
}
