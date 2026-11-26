package com.gestaoretalho.pos

import android.content.Context
import com.sunmi.peripheral.printer.InnerPrinterCallback
import com.sunmi.peripheral.printer.InnerPrinterManager
import com.sunmi.peripheral.printer.SunmiPrinterService
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/**
 * Impressora térmica embutida do Sunmi (58 mm, 32 colunas), pelo serviço
 * do sistema (woyou.aidlservice.jiuiv5) através da biblioteca da Sunmi.
 * Liga-se à primeira impressão e mantém a ligação.
 */
object ImpressoraSunmi {
    @Volatile private var servico: SunmiPrinterService? = null

    private fun ligar(contexto: Context): SunmiPrinterService {
        servico?.let { return it }
        val pronto = CountDownLatch(1)
        val ok = try {
            InnerPrinterManager.getInstance().bindService(contexto.applicationContext, object : InnerPrinterCallback() {
                override fun onConnected(s: SunmiPrinterService) {
                    servico = s
                    pronto.countDown()
                }

                override fun onDisconnected() {
                    servico = null
                }
            })
        } catch (e: Exception) {
            false
        }
        if (!ok) throw IllegalStateException("este aparelho não tem impressora embutida")
        pronto.await(5, TimeUnit.SECONDS)
        return servico ?: throw IllegalStateException("a impressora do aparelho não respondeu")
    }

    fun imprimir(contexto: Context, texto: String) {
        val s = ligar(contexto)
        // 1 = pronta; 2..7 = a iniciar, erro, sem papel, sobreaquecida, tampa aberta…
        when (s.updatePrinterState()) {
            1, 2 -> {}
            4 -> throw IllegalStateException("a impressora do aparelho está sem papel")
            5 -> throw IllegalStateException("a impressora do aparelho está sobreaquecida")
            6 -> throw IllegalStateException("a tampa da impressora está aberta")
            else -> {}
        }
        s.printerInit(null)
        s.setFontSize(24f, null) // 24 px → 32 caracteres no papel de 58 mm
        s.printText(if (texto.endsWith("\n")) texto else texto + "\n", null)
        s.lineWrap(4, null)
    }
}
