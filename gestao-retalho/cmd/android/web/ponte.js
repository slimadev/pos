// Ponte REAL: cria window.go.admin.AdminApp e window.go.api.App com a mesma
// forma que o Wails cria no desktop, mas cada chamada vai para o Kotlin
// (window.Ponte.chamar) e daí para o backend Go (gobridge). A resposta volta
// assíncrona por window.__ponte.resolver. Escrito em ES5 (WebView Chrome 62).
(function () {
  "use strict";

  window.__PLATAFORMA = "android";

  var pendentes = {};
  var proximo = 1;

  window.__ponte = {
    // ok=true: `dados` é o resultado em JSON; ok=false: a mensagem de erro
    // (o Wails rejeita a Promise com a mensagem, e os ecrãs fazem String(e)).
    resolver: function (id, ok, dados) {
      var p = pendentes[id];
      if (!p) return;
      delete pendentes[id];
      if (!ok) {
        p.reject(dados);
        return;
      }
      try {
        p.resolve(dados === "" ? null : JSON.parse(dados));
      } catch (e) {
        p.reject("Resposta inválida do backend: " + e.message);
      }
    },
  };

  function criarAlvo(alvo) {
    return new Proxy({}, {
      get: function (_, metodo) {
        metodo = String(metodo);
        return function () {
          var args = Array.prototype.slice.call(arguments);
          // O PDV é outro ecrã Android (não um processo como no desktop).
          if (alvo === "admin.AdminApp" && metodo === "AbrirPDV") {
            window.Nativo.abrirPDV();
            return Promise.resolve(null);
          }
          return new Promise(function (resolve, reject) {
            var id = proximo++;
            pendentes[id] = { resolve: resolve, reject: reject };
            window.Ponte.chamar(id, alvo, metodo, JSON.stringify(args));
          });
        };
      },
    });
  }

  window.go = {
    admin: { AdminApp: criarAlvo("admin.AdminApp") },
    api: { App: criarAlvo("api.App") },
  };

  // Leitor de códigos em modo "teclado" (alternativa ao broadcast, que o
  // Kotlin já converte em `codigo-lido`): o leitor "escreve" o código muito
  // depressa e termina com Enter. Fora de um campo de texto isso perdia-se;
  // aqui vira o mesmo evento `codigo-lido`. Dentro de um campo, o próprio
  // campo trata (ex: a pesquisa do PDV junta o produto com Enter).
  var tecladas = "";
  var ultimaTecla = 0;
  window.addEventListener(
    "keydown",
    function (e) {
      var tag = e.target && e.target.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      var agora = Date.now();
      if (agora - ultimaTecla > 80) tecladas = ""; // uma pessoa não escreve tão depressa
      ultimaTecla = agora;
      if (e.key === "Enter") {
        if (tecladas.length >= 4) {
          e.preventDefault();
          window.dispatchEvent(new CustomEvent("codigo-lido", { detail: tecladas }));
        }
        tecladas = "";
      } else if (e.key && e.key.length === 1) {
        tecladas += e.key;
      }
    },
    true,
  );
})();
