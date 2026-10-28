package com.gestaoretalho.pos

import android.app.Activity
import android.app.Application
import android.os.Bundle
import android.util.Log
import com.gestaoretalho.gobridge.Gobridge
import java.util.TimeZone

/**
 * Arranca o backend Go (gobridge) uma única vez por processo, antes de
 * qualquer ecrã: dados na pasta privada da app e o fuso horário do
 * terminal. Também guarda a Activity visível, de que a PlataformaAndroid
 * precisa para abrir ficheiros e o seletor.
 */
class RetalhoApp : Application() {
    override fun onCreate() {
        super.onCreate()
        registerActivityLifecycleCallbacks(object : ActivityLifecycleCallbacks {
            override fun onActivityResumed(a: Activity) { atividadeAtual = a as? WebActivity }
            override fun onActivityPaused(a: Activity) { if (atividadeAtual === a) atividadeAtual = null }
            override fun onActivityCreated(a: Activity, s: Bundle?) {}
            override fun onActivityStarted(a: Activity) {}
            override fun onActivityStopped(a: Activity) {}
            override fun onActivitySaveInstanceState(a: Activity, s: Bundle) {}
            override fun onActivityDestroyed(a: Activity) {}
        })

        // O fuso vai pelo nome (ex: "Africa/Maputo"); o Go aplica-o a si e ao
        // SQLite (ver gobridge/fuso_linux.go).
        erroArranque = try {
            Gobridge.iniciar(filesDir.absolutePath, TimeZone.getDefault().id, PlataformaAndroid(this))
            null
        } catch (e: Exception) {
            Log.e("Retalho", "backend não arrancou", e)
            e.message ?: e.toString()
        }
    }

    companion object {
        @Volatile var atividadeAtual: WebActivity? = null
        /** Mensagem do erro de arranque do backend (null = arrancou bem). */
        @Volatile var erroArranque: String? = null
    }
}
