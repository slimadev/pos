package com.gestaoretalho.pos

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.webkit.MimeTypeMap
import androidx.core.content.FileProvider
import com.gestaoretalho.gobridge.Plataforma
import java.io.File
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/**
 * O que no desktop o sistema operativo/Wails fazem e no Android tem de ser a
 * app nativa: abrir um PDF/Excel gerado e escolher um ficheiro (importação
 * de produtos). Chamado pelo Go, numa thread de fundo.
 */
class PlataformaAndroid(private val contexto: Context) : Plataforma {

    override fun abrirFicheiro(caminho: String) {
        val atividade = RetalhoApp.atividadeAtual ?: throw IllegalStateException("nenhum ecrã ativo")
        val ficheiro = File(caminho)
        val uri = FileProvider.getUriForFile(contexto, "${contexto.packageName}.ficheiros", ficheiro)
        val mime = MimeTypeMap.getSingleton().getMimeTypeFromExtension(ficheiro.extension.lowercase())
            ?: "application/octet-stream"
        atividade.runOnUiThread {
            val ver = Intent(Intent.ACTION_VIEW).setDataAndType(uri, mime)
                .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            try {
                atividade.startActivity(ver)
            } catch (e: ActivityNotFoundException) {
                // Sem leitor de PDF/Excel no terminal: partilhar (WhatsApp, email…).
                val partilhar = Intent(Intent.ACTION_SEND).setType(mime)
                    .putExtra(Intent.EXTRA_STREAM, uri)
                    .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                atividade.startActivity(Intent.createChooser(partilhar, ficheiro.name))
            }
        }
    }

    override fun escolherFicheiro(titulo: String, extensoes: String): String {
        val atividade = RetalhoApp.atividadeAtual ?: throw IllegalStateException("nenhum ecrã ativo")
        val fim = CountDownLatch(1)
        var resultado = ""
        atividade.runOnUiThread {
            atividade.escolherFicheiro(titulo, extensoes) { caminho ->
                resultado = caminho
                fim.countDown()
            }
        }
        // O Go espera (numa thread de fundo) até o utilizador escolher ou cancelar.
        fim.await(10, TimeUnit.MINUTES)
        return resultado
    }
}
